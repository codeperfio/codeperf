package cmd

import (
	"github.com/google/pprof/profile"
	"log"
	"regexp"
	"sort"
	"strings"
)

type treeNode struct {
	Name     string               `json:"n"`
	FullName string               `json:"f"`
	Cum      int64                `json:"v"`
	Children map[string]*treeNode `json:"c"`
}

type treeNodeSlice struct {
	Name     string          `json:"n"`
	FullName string          `json:"f"`
	Cum      int64           `json:"v"`
	Children []treeNodeSlice `json:"c"`
}

var (
	// Removes package name and method arguments for Java method names.
	// See tests for examples.
	javaRegExp = regexp.MustCompile(`^(?:[a-z]\w*\.)*([A-Z][\w\$]*\.(?:<init>|[a-z][\w\$]*(?:\$\d+)?))(?:(?:\()|$)`)
	// Removes package name and method arguments for Go function names.
	// See tests for examples.
	goRegExp = regexp.MustCompile(`^(?:[\w\-\.]+\/)+(.+)`)
	// Removes potential module versions in a package path.
	goVerRegExp = regexp.MustCompile(`^(.*?)/v(?:[2-9]|[1-9][0-9]+)([./].*)$`)
	// Strips C++ namespace prefix from a C++ function / method name.
	// NOTE: Make sure to keep the template parameters in the name. Normally,
	// template parameters are stripped from the C++ names but when
	// -symbolize=demangle=templates flag is used, they will not be.
	// See tests for examples.
	cppRegExp                = regexp.MustCompile(`^(?:[_a-zA-Z]\w*::)+(_*[A-Z]\w*::~?[_a-zA-Z]\w*(?:<.*>)?)`)
	cppAnonymousPrefixRegExp = regexp.MustCompile(`^\(anonymous namespace\)::`)
)

// ShortenFunctionName returns a shortened version of a function's name.
func ShortenFunctionName(f string) string {
	f = cppAnonymousPrefixRegExp.ReplaceAllString(f, "")
	f = goVerRegExp.ReplaceAllString(f, `${1}${2}`)
	for _, re := range []*regexp.Regexp{goRegExp, javaRegExp, cppRegExp} {
		if matches := re.FindStringSubmatch(f); len(matches) >= 2 {
			return strings.Join(matches[1:], "")
		}
	}
	return f
}

// Convert marshals the given protobuf profile into folded text format.
func profileToFolded(protobuf *profile.Profile) treeNodeSlice {
	rootNode := treeNode{"root", "root", 0, make(map[string]*treeNode, 0)}
	if err := protobuf.Aggregate(true, true, false, false, false); err != nil {
		log.Fatal(err)
	}
	protobuf = protobuf.Compact()
	sort.Slice(protobuf.Sample, func(i, j int) bool {
		return protobuf.Sample[i].Value[0] > protobuf.Sample[j].Value[0]
	})

	for _, sample := range protobuf.Sample {
		var cum int64 = 0
		for _, val := range sample.Value {
			cum = cum + val
			break
		}
		var frames []string
		var currentNode *treeNode
		var currentMap map[string]*treeNode = rootNode.Children
		for i := range sample.Location {
			var ok bool
			loc := sample.Location[len(sample.Location)-i-1]
			for j := range loc.Line {
				line := loc.Line[len(loc.Line)-j-1]
				fname := ShortenFunctionName(line.Function.Name)
				shortName := fname[strings.LastIndex(fname, ".")+1:]
				currentNode, ok = currentMap[shortName]
				if !ok {
					currentNode = &treeNode{shortName, fname, 0, make(map[string]*treeNode, 0)}
					currentMap[shortName] = currentNode
				}
				currentNode.Cum += cum
				currentMap = currentNode.Children
				frames = append(frames, shortName)
			}
		}
	}
	finalTree := treeNodeSlice{rootNode.Name, rootNode.FullName, rootNode.Cum, collapse(rootNode.Children)}
	return finalTree
}

func collapse(children map[string]*treeNode) (tree []treeNodeSlice) {
	tree = make([]treeNodeSlice, 0)
	for _, k := range children {
		nS := treeNodeSlice{k.Name, k.FullName, k.Cum, collapse(k.Children)}
		tree = append(tree, nS)
	}
	return
}
