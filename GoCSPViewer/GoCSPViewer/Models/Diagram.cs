using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.Threading.Tasks;

namespace GoCSPViewer.Models
{
    public class DiagramModel
    {
        public List<NodeModel> Nodes { get; set; } = [];
        public List<EdgeModel> Edges { get; set; } = [];
    }

    public class NodeModel
    {
        public string Id { get; set; } = "";
        public string Type { get; set; } = "";
        public string? Operation { get; set; }
        public List<string>? Inputs { get; set; }
        public List<string>? Outputs { get; set; }
    }

    public class EdgeModel
    {
        public string From { get; set; } = "";
        public string To { get; set; } = "";
    }
}
