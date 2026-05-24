using System.Text.Json.Serialization;

namespace GoCSPViewer.Models;

public class EventMessage
{
    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("event")]
    public string Event { get; set; }

    [JsonPropertyName("node")]
    public string Node { get; set; }

    [JsonPropertyName("port")]
    public string Port { get; set; }

    [JsonPropertyName("value")]
    public object Value { get; set; }
}