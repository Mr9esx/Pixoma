package comfyui

import (
	"encoding/json"
	"fmt"
)

// NodeImage is one binary file recorded for a node, with Comfy view metadata.
// ComfyUI reports video and audio files through separate history arrays, so
// this type represents those output files as well.
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

type historyFile struct {
	Filename  string `json:"filename"`
	Subfolder string `json:"subfolder"`
	Type      string `json:"type"`
}

type historyNodeOutput struct {
	Images []historyFile `json:"images"`
	Audio  []historyFile `json:"audio"`
	Gifs   []historyFile `json:"gifs"`
	Text   []string      `json:"text"`
}

type historyEntry struct {
	Outputs map[string]historyNodeOutput `json:"outputs"`
	Status  struct {
		StatusStr string `json:"status_str"`
		Completed bool   `json:"completed"`
	} `json:"status"`
}

func appendHistoryFiles(dst []NodeImage, files []historyFile) []NodeImage {
	for _, file := range files {
		dst = append(dst, NodeImage{
			OutputFile: OutputFile{Filename: file.Filename},
			Subfolder:  file.Subfolder,
			Type:       file.Type,
		})
	}
	return dst
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
			images := appendHistoryFiles(nil, node.Images)
			images = appendHistoryFiles(images, node.Audio)
			images = appendHistoryFiles(images, node.Gifs)
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
