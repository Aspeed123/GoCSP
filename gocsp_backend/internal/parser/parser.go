package parser

import (
    "encoding/json"
    "os"

    "gocsp/internal/model"
)

func LoadDiagram(path string) (*model.Diagram, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }

    var d model.Diagram

    err = json.Unmarshal(data, &d)
    if err != nil {
        return nil, err
    }

    return &d, nil
}