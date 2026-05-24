using System.IO;
using System.Text.Json;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Media;
using System.Windows.Forms.Integration;
using Microsoft.Msagl.Drawing;
using Microsoft.Msagl.Layout.Layered;
using Microsoft.Msagl.WpfGraphControl;
using Microsoft.Msagl.GraphViewerGdi;

using Websocket.Client;

using GoCSPViewer.Models;

namespace GoCSPViewer;

public partial class MainWindow : Window
{
    private WebsocketClient? _client;

    private DiagramModel? _diagram;

    private Graph? _graph;

    private GViewer? _viewer;

    private Dictionary<string, Microsoft.Msagl.Drawing.Color> _nodeBaseColors = new();

    private Dictionary<string, List<Edge>> _outgoingEdges = new();

    private Dictionary<string, List<Edge>> _portEdges = new();

    private readonly SemaphoreSlim _animationLock = new(1, 1);

    private bool _sequentialMode = false;

    private double _playbackSpeed = 1.0;

    public MainWindow()
    {
        InitializeComponent();

        LoadDiagram();
    }

    private void LoadDiagram()
    {
        var path = Path.GetFullPath(
            Path.Combine(
                AppDomain.CurrentDomain.BaseDirectory,
                "..", "..", "..", "..", "..",
                "examples",
                "pipeline.json"
            )
        );

        var json = File.ReadAllText(path);

        _diagram = JsonSerializer.Deserialize<DiagramModel>(
            json,
            new JsonSerializerOptions { PropertyNameCaseInsensitive = true });

        if (_diagram == null)
            return;

        // MSAGL graph
        _graph = new Graph();

        // nodes
        foreach (var node in _diagram.Nodes)
        {
            var n = _graph.AddNode(node.Id);
            n.LabelText = node.Id;
            var baseColor = node.Type switch
            {
                "generator" => Microsoft.Msagl.Drawing.Color.LightSteelBlue,
                "processor" => Microsoft.Msagl.Drawing.Color.Gainsboro,
                "sink" => Microsoft.Msagl.Drawing.Color.PaleGreen,
                _ => Microsoft.Msagl.Drawing.Color.Gray,
            };
            n.Attr.FillColor = baseColor;

            _nodeBaseColors[node.Id] = baseColor;
        }

        // edges
        foreach (var edge in _diagram.Edges)
        {
            var from = edge.From.Split('.')[0];
            var to = edge.To.Split('.')[0];

            var edgeObj = _graph.AddEdge(from, to);
            edgeObj.LabelText = "";

            var fromPort =
                $"{from}.{edge.From.Split('.')[1]}";

            var toPort =
                $"{to}.{edge.To.Split('.')[1]}";

            if (!_portEdges.ContainsKey(fromPort))
            {
                _portEdges[fromPort] =
                    new List<Edge>();
            }

            if (!_portEdges.ContainsKey(toPort))
            {
                _portEdges[toPort] =
                    new List<Edge>();
            }

            _portEdges[fromPort].Add(edgeObj);
            _portEdges[toPort].Add(edgeObj);

            if (!_outgoingEdges.ContainsKey(from))
            {
                _outgoingEdges[from] = new List<Edge>();
            }

            _outgoingEdges[from].Add(edgeObj);
        }

        _graph.Attr.LayerDirection = LayerDirection.LR;

        RenderGraph();
    }

    private void RenderGraph()
    {
        _viewer = new GViewer
        {
            Graph = _graph,
            ToolBarIsVisible = false
        };

        var host = new WindowsFormsHost
        {
            Child = _viewer
        };

        MainCanvas.Children.Clear();
        MainCanvas.Children.Add(host);
    }

    private void ConnectWebSocket()
    {
        var url =
            new Uri("ws://localhost:8080/ws");

        _client =
            new WebsocketClient(url)
            {
                IsReconnectionEnabled = false
            };

        _client.MessageReceived.Subscribe(msg =>
        {
            if (string.IsNullOrWhiteSpace(msg.Text))
                return;

            var eventMessage =
                JsonSerializer.Deserialize<EventMessage>(
                    msg.Text
                );

            if (eventMessage == null)
                return;

            Dispatcher.InvokeAsync(async () =>
            {
                await HandleEvent(eventMessage);
            });
        });

        _client.Start();
    }

    private async Task HandleEvent(EventMessage evt)
    {
        if (_sequentialMode)
        {
            await _animationLock.WaitAsync();

            try
            {
                await HandleEventInternal(evt);
            }
            finally
            {
                _animationLock.Release();
            }
        }
        else
        {
            await HandleEventInternal(evt);
        }
    }

    private async Task HandleEventInternal(EventMessage evt)
    {
        var node = _graph?.FindNode(evt.Node);
        if (node == null || _viewer == null)
            return;

        var color = GetEventColor(evt.Event);
        var originalNodeColor = _nodeBaseColors[evt.Node];

        // 1. ПОДЦВЕТКА УЗЛА
        if (evt.Event == "port_closed")
        {
            node.Attr.Color = Microsoft.Msagl.Drawing.Color.Red;
            node.Attr.LineWidth = 3;
        }
        else
        {
            node.Attr.FillColor = color;
        }

        // 2. ПОИСК СВЯЗАННЫХ РЕБЕР
        List<Edge> affectedEdges = new();
        var portKey = $"{evt.Node}.{evt.Port}";

        if (evt.Event == "send" && _outgoingEdges.TryGetValue(evt.Node, out var outEdges))
        {
            affectedEdges.AddRange(outEdges);
        }
        else if ((evt.Event == "port_closed" || evt.Event == "initial_token") &&
                 _portEdges.TryGetValue(portKey, out var portEdges))
        {
            affectedEdges.AddRange(portEdges);
        }

        // 3. ПРИМЕНЕНИЕ СТИЛЯ К РЕБРАМ
        bool isDataEvent = evt.Event == "send" || evt.Event == "initial_token";
        foreach (var edge in affectedEdges)
        {
            edge.Attr.Color = color;
            edge.Attr.LineWidth = 3;

            if (isDataEvent)
            {
                edge.LabelText = evt.Value?.ToString();
            }
        }

        _viewer.Invalidate();

        // 4. ТАЙМИНГИ АНИМАЦИИ
        var totalDelay = (int)(1000 / _playbackSpeed);
        var flashDelay = (int)(totalDelay * 0.8);
        var restDelay = totalDelay - flashDelay;

        await Task.Delay(flashDelay);

        // 5. ВОССТАНОВЛЕНИЕ ИСХОДНОГО СОСТОЯНИЯ
        node.Attr.FillColor = originalNodeColor;
        node.Attr.Color = Microsoft.Msagl.Drawing.Color.Black;
        node.Attr.LineWidth = 1;

        foreach (var edge in affectedEdges)
        {
            edge.Attr.Color = Microsoft.Msagl.Drawing.Color.Black;
            edge.Attr.LineWidth = 1;
            edge.LabelText = "";
        }

        _viewer.Invalidate();

        if (restDelay > 0)
        {
            await Task.Delay(restDelay);
        }
    }

    private Microsoft.Msagl.Drawing.Color GetEventColor(string evt)
    {
        return evt switch
        {
            // goroutine lifecycle
            "goroutine_start" =>
                Microsoft.Msagl.Drawing.Color.MediumPurple,

            // data flow
            "initial_token" => Microsoft.Msagl.Drawing.Color.MediumSeaGreen,

            "send" => Microsoft.Msagl.Drawing.Color.DeepSkyBlue,

            "receive" => Microsoft.Msagl.Drawing.Color.Orange,

            // shutdown / close
            "port_closed" => Microsoft.Msagl.Drawing.Color.Red,

            "node_stop" => Microsoft.Msagl.Drawing.Color.DarkRed,

            _ => Microsoft.Msagl.Drawing.Color.Yellow
        };
    }

    private void ModeButton_Click(
        object sender,
        RoutedEventArgs e)
    {
        _sequentialMode =
            !_sequentialMode;

        ModeButton.Content =
            _sequentialMode
                ? "Mode: Sequential"
                : "Mode: Async";
    }

    private void RunButton_Click(
        object sender,
        RoutedEventArgs e)
    {
        ConnectWebSocket();
    }

    private void SpeedSlider_ValueChanged(
        object sender,
        RoutedPropertyChangedEventArgs<double> e)
    {
        _playbackSpeed = e.NewValue;

        if (SpeedLabel != null)
        {
            SpeedLabel.Text =
                $"{_playbackSpeed:0.##}x";
        }
    }
}