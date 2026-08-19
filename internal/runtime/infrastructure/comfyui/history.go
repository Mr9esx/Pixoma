package comfyui

import (
	"encoding/json"
	"fmt"
)

// NodeImage is one image recorded for a node, with Comfy view metadata.
type NodeImage struct {
	OutputFile
	Subfolder string
	Type      string
}

// NodeOutput groups everything ComfyUI recorded for one node in a prompt history entry.
type NodeOutput struct {
	Images []NodeImage
	Texts  []string
}

// HistoryResult maps node IDs to their outputs for one prompt.
type HistoryResult map[string]NodeOutput

type historyEntry struct {
	Outputs map[string]struct {
		Images []struct {
			Filename  string `json:"filename"`
			Subfolder string `json:"subfolder"`
			Type      string `json:"type"`
		} `json:"images"`
		Text []string `json:"text"`
	} `json:"outputs"`
	Status struct {
		StatusStr string `json:"status_str"`
		Completed bool   `json:"completed"`
	} `json:"status"`
}

// ParseHistory parses a GET /history/{prompt_id} response body.
func ParseHistory(raw []byte) (HistoryResult, error) {
	var hist map[string]historyEntry
	if err := json.Unmarshal(raw, &hist); err != nil {
		return nil, fmt.Errorf("comfyui history decode: %w", err)
	}
	out := HistoryResult{}
	for _, entry := range hist {
		for nodeID, node := range entry.Outputs {
			var images []NodeImage
			for _, img := range node.Images {
				images = append(images, NodeImage{
					OutputFile: OutputFile{Filename: img.Filename},
					Subfolder:  img.Subfolder,
					Type:       img.Type,
				})
			}
			if len(images) == 0 && len(node.Text) == 0 {
				continue
			}
			out[nodeID] = NodeOutput{
				Images: images,
				Texts:  append([]string(nil), node.Text...),
			}
		}
	}
	return out, nil
}
