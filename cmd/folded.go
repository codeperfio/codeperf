package cmd

import (
	"fmt"
	"github.com/google/pprof/profile"
	"log"
	"sort"
	"strings"
)

type treeNode struct {
	Name     string               `json:"n"`
	Cum      int64                `json:"v"`
	Children map[string]*treeNode `json:"c"`
}

type treeNodeSlice struct {
	Name     string          `json:"n"`
	Cum      int64           `json:"v"`
	Children []treeNodeSlice `json:"c"`
}

// Convert marshals the given protobuf profile into folded text format.
func profileToFolded(protobuf *profile.Profile) treeNodeSlice {
	rootNode := treeNode{"root", 0, make(map[string]*treeNode, 0)}
	if err := protobuf.Aggregate(true, true, false, false, false); err != nil {
		log.Fatal(err)
	}
	protobuf = protobuf.Compact()
	sort.Slice(protobuf.Sample, func(i, j int) bool {
		return protobuf.Sample[i].Value[0] > protobuf.Sample[j].Value[0]
	})
	for _, sample := range protobuf.Sample {
		var frames []string
		var currentNode *treeNode
		var currentMap map[string]*treeNode = rootNode.Children
		for i := range sample.Location {
			var ok bool
			loc := sample.Location[len(sample.Location)-i-1]
			for j := range loc.Line {
				line := loc.Line[len(loc.Line)-j-1]
				fname := line.Function.Name
				currentNode, ok = currentMap[fname]
				if !ok {
					currentNode = &treeNode{fname, 0, make(map[string]*treeNode, 0)}
					currentMap[fname] = currentNode
				}
				currentMap = currentNode.Children
				frames = append(frames, fname)
			}
		}

		var values []string
		for _, val := range sample.Value {
			values = append(values, fmt.Sprintf("%d", val))
			currentNode.Cum = currentNode.Cum + val
			break
		}
		fmt.Printf(
			"%s %s\n",
			strings.Join(frames, ";"),
			strings.Join(values, " "),
		)
	}
	finalTree := treeNodeSlice{rootNode.Name, rootNode.Cum, collapse(rootNode.Children)}
	return finalTree
}

func collapse(children map[string]*treeNode) (tree []treeNodeSlice) {
	tree = make([]treeNodeSlice, 0)
	for _, k := range children {
		nS := treeNodeSlice{k.Name, k.Cum, collapse(k.Children)}
		tree = append(tree, nS)
	}
	return
}
