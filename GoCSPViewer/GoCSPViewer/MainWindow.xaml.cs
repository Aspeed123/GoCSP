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
using System.Diagnostics;

using Websocket.Client;

using GoCSPViewer.Models;
using Microsoft.Win32;

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

    private DateTimeOffset? _lastEventTimestamp = null;

    private double _playbackSpeed = 1.0;

    private Process? _gocspProcess;

    private Process? _serverProcess;

    public MainWindow()
    {
        InitializeComponent();

        OpenDiagram();
    }

    private void OpenDiagram()
    {
        var path = SelectDiagramFile();

        if (path == null)
        {
            this.Close();
            return;
        }

        ExecuteGoBackend(path);

        LoadDiagramFromFile(path);

        BuildGraph();

        RenderGraph();
    }

    private void ExecuteGoBackend(string diagramPath)
    {
        try
        {
            string projectRoot = Path.GetFullPath(
                Path.Combine(AppDomain.CurrentDomain.BaseDirectory, "..", "..", "..", "..", "..", "gocsp")
            );
            string relativeJsonPath = Path.GetRelativePath(projectRoot, diagramPath);

            ProcessStartInfo gocspInfo = new()
            {
                FileName = "go",
                Arguments = $"run ./cmd/gocsp \"{relativeJsonPath}\"",
                WorkingDirectory = projectRoot,
                CreateNoWindow = true,
                UseShellExecute = false,
                RedirectStandardInput = true
            };

            _gocspProcess = Process.Start(gocspInfo);


            ProcessStartInfo serverInfo = new()
            {
                FileName = "go",
                Arguments = "run ./cmd/replay_server",
                WorkingDirectory = projectRoot,
                CreateNoWindow = true,
                UseShellExecute = false
            };

            _serverProcess = Process.Start(serverInfo);
        }
        catch (Exception ex)
        {
            MessageBox.Show($"Ошибка при запуске Go-сервера: {ex.Message}", "Ошибка инициализации", MessageBoxButton.OK, MessageBoxImage.Error);
        }
    }

    protected override void OnClosed(EventArgs e)
    {
        base.OnClosed(e);

        if (_gocspProcess != null && !_gocspProcess.HasExited)
        {
            try
            {
                _gocspProcess.Kill(entireProcessTree: true);
                _gocspProcess.Dispose();
            }
            catch { }
        }

        if (_serverProcess != null && !_serverProcess.HasExited)
        {
            try
            {
                _serverProcess.Kill(entireProcessTree: true);
                _serverProcess.Dispose();
            }
            catch { }
        }
    }

    private string? SelectDiagramFile()
    {
        var defaultDir = Path.GetFullPath(
            Path.Combine(
                AppDomain.CurrentDomain.BaseDirectory,
                "..", "..", "..", "..", "..",
                "examples"
            )
        );

        OpenFileDialog openFileDialog = new()
        {
            Title = "Выберите файл диаграммы",
            InitialDirectory = Directory.Exists(defaultDir) ? defaultDir : "c:\\",

            Filter = "JSON files (*.json)|*.json|All files (*.*)|*.*",
            FilterIndex = 1,
            RestoreDirectory = true
        };

        return openFileDialog.ShowDialog() == true ? openFileDialog.FileName : null;
    }

    private void LoadDiagramFromFile(string path)
    {
        var json = File.ReadAllText(path);

        _diagram = JsonSerializer.Deserialize<DiagramModel>(
            json,
            new JsonSerializerOptions { PropertyNameCaseInsensitive = true });
    }

    private void BuildGraph()
    {
        if (_diagram == null)
            return;

        _graph = new Graph();
        _portEdges.Clear();
        _outgoingEdges.Clear();
        _nodeBaseColors.Clear();

        // Наполнение узлов
        foreach (var node in _diagram.Nodes)
        {
            var n = _graph.AddNode(node.Id);
            n.LabelText = node.Id;

            var baseColor = node.Type switch
            {
                "generator" => Microsoft.Msagl.Drawing.Color.LightSteelBlue,
                "processor" => Microsoft.Msagl.Drawing.Color.Gainsboro,
                "switch" => Microsoft.Msagl.Drawing.Color.Orchid,
                "sink" => Microsoft.Msagl.Drawing.Color.PaleGreen,
                "merge" => Microsoft.Msagl.Drawing.Color.Moccasin,
                _ => Microsoft.Msagl.Drawing.Color.Gray,
            };
            n.Attr.FillColor = baseColor;
            _nodeBaseColors[node.Id] = baseColor;
        }

        // Наполнение ребер
        foreach (var edge in _diagram.Edges)
        {
            var from = edge.From.Split('.')[0];
            var to = edge.To.Split('.')[0];

            var edgeObj = _graph.AddEdge(from, to);
            edgeObj.LabelText = "";

            string fromPort = edge.From;
            string toPort = edge.To;

            if (!_portEdges.TryGetValue(fromPort, out var fromList))
                _portEdges[fromPort] = fromList = new List<Edge>();
            if (!_portEdges.TryGetValue(toPort, out var toList))
                _portEdges[toPort] = toList = new List<Edge>();
            if (!_outgoingEdges.TryGetValue(from, out var outList))
                _outgoingEdges[from] = outList = new List<Edge>();

            fromList.Add(edgeObj);
            toList.Add(edgeObj);
            outList.Add(edgeObj);
        }

        _graph.Attr.LayerDirection = LayerDirection.LR;
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

        if (evt.Event == "port_closed")
        {
            node.Attr.Color = Microsoft.Msagl.Drawing.Color.Red;
            node.Attr.LineWidth = 3;
        }
        else
        {
            node.Attr.FillColor = color;
        }

        List<Edge> affectedEdges = new();
        var portKey = $"{evt.Node}.{evt.Port}";

        if (evt.Event == "send")
        {
            if (!string.IsNullOrEmpty(evt.Port) && _portEdges.TryGetValue(portKey, out var portEdges))
            {
                affectedEdges.AddRange(portEdges);
            }
            else if (_outgoingEdges.TryGetValue(evt.Node, out var outEdges))
            {
                affectedEdges.AddRange(outEdges);
            }
        }
        else if ((evt.Event == "port_closed" || evt.Event == "initial_token") &&
                 _portEdges.TryGetValue(portKey, out var portEdges))
        {
            affectedEdges.AddRange(portEdges);
        }

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

        int baseFlashMs = 200;

        int flashDelay = (int)(baseFlashMs / _playbackSpeed);

        if (flashDelay < 30) flashDelay = 30;

        int totalWaitTime = flashDelay;

        if (evt.Event == "port_closed" || evt.Event == "node_stop")
        {
            totalWaitTime = flashDelay * 2;
            _lastEventTimestamp = evt.Time;
        }
        else
        {
            if (_lastEventTimestamp.HasValue)
            {
                TimeSpan timeDifference = evt.Time - _lastEventTimestamp.Value;
                int realDeltaMs = (int)(timeDifference.TotalMilliseconds / _playbackSpeed);

                int maxPauseMs = (int)(1500 / _playbackSpeed);
                if (realDeltaMs > maxPauseMs) realDeltaMs = maxPauseMs;

                totalWaitTime = realDeltaMs;
            }

            _lastEventTimestamp = evt.Time;
        }

        int restDelay = totalWaitTime - flashDelay;

        int minRestMs = (int)(50 / _playbackSpeed);
        if (minRestMs < 16) minRestMs = 16;

        if (restDelay < minRestMs)
        {
            restDelay = minRestMs;
        }

        await Task.Delay(flashDelay);

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
            "goroutine_start" => Microsoft.Msagl.Drawing.Color.MediumPurple,

            "initial_token" => Microsoft.Msagl.Drawing.Color.MediumSeaGreen,

            "send" => Microsoft.Msagl.Drawing.Color.DeepSkyBlue,

            "receive" => Microsoft.Msagl.Drawing.Color.Orange,

            "dropped" => Microsoft.Msagl.Drawing.Color.DimGray,

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
        if (_gocspProcess != null && !_gocspProcess.HasExited)
        {
            _gocspProcess.StandardInput.WriteLine("stop");
        }

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