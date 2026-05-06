package dim

import (
	"sort"
	"strings"
)

// nodeTyp represents the type of a radix tree node.
type nodeTyp uint8

const (
	ntStatic  nodeTyp = iota // /users/
	ntParam                  // {id}
	ntCatchAll               // {path...}
)

// treeEndpoint holds the handler for a specific HTTP method.
type treeEndpoint struct {
	handler HandlerFunc
}

// treeNode is a node in the radix tree.
// Static children are stored as a compressed prefix (radix compression).
// children[0] = static, children[1] = param, children[2] = catchall.
type treeNode struct {
	prefix    string
	label     byte // first byte of prefix for O(1) label comparison
	typ       nodeTyp
	paramKey  string                    // key name for ntParam / ntCatchAll
	endpoints map[string]*treeEndpoint  // method → endpoint
	children  [3][]*treeNode
}

func newTreeNode(typ nodeTyp, prefix string) *treeNode {
	n := &treeNode{
		typ:       typ,
		prefix:    prefix,
		endpoints: make(map[string]*treeEndpoint),
	}
	if len(prefix) > 0 {
		n.label = prefix[0]
	}
	return n
}

// insert adds a route pattern + method + handler into the subtree rooted at n.
func (n *treeNode) insert(pattern, method string, handler HandlerFunc) {
	if pattern == "" {
		n.endpoints[method] = &treeEndpoint{handler: handler}
		return
	}
	if pattern[0] == '{' {
		n.insertParam(pattern, method, handler)
		return
	}
	n.insertStatic(pattern, method, handler)
}

// insertStatic handles a static prefix segment during insert.
func (n *treeNode) insertStatic(pattern, method string, handler HandlerFunc) {
	// Identify the static portion: everything up to the first '{'.
	end := strings.IndexByte(pattern, '{')
	staticPart := pattern
	if end >= 0 {
		staticPart = pattern[:end]
	}

	label := staticPart[0]

	for i, c := range n.children[ntStatic] {
		if c.label != label {
			continue
		}

		lcp := longestCommonPrefix(staticPart, c.prefix)

		if lcp == len(c.prefix) {
			// Existing prefix fully consumed — recurse with remainder of pattern.
			c.insert(pattern[lcp:], method, handler)
			return
		}

		// Partial match: split the existing node at lcp.
		//   Before: n → c (prefix=c.prefix, children=c.children)
		//   After:  n → inter (prefix=c.prefix[:lcp]) → c (prefix=c.prefix[lcp:])
		inter := newTreeNode(ntStatic, c.prefix[:lcp])
		c.prefix = c.prefix[lcp:]
		c.label = c.prefix[0]
		inter.children[ntStatic] = []*treeNode{c}
		n.children[ntStatic][i] = inter

		// Continue insert of the remaining new pattern into the split node.
		inter.insert(pattern[lcp:], method, handler)
		return
	}

	// No compatible child found — create a new static node.
	child := newTreeNode(ntStatic, staticPart)
	n.children[ntStatic] = append(n.children[ntStatic], child)
	child.insert(pattern[len(staticPart):], method, handler)
}

// insertParam handles a '{...}' segment during insert.
func (n *treeNode) insertParam(pattern, method string, handler HandlerFunc) {
	end := strings.IndexByte(pattern, '}')
	if end < 0 {
		panic("dim: malformed route pattern — missing '}'")
	}

	key := pattern[1:end]
	isCatchAll := strings.HasSuffix(key, "...")
	if isCatchAll {
		key = strings.TrimSuffix(key, "...")
	}

	childTyp := ntParam
	if isCatchAll {
		childTyp = ntCatchAll
	}

	// Reuse an existing child with the same key.
	for _, c := range n.children[childTyp] {
		if c.paramKey == key {
			c.insert(pattern[end+1:], method, handler)
			return
		}
	}

	child := newTreeNode(childTyp, pattern[:end+1])
	child.paramKey = key
	n.children[childTyp] = append(n.children[childTyp], child)
	child.insert(pattern[end+1:], method, handler)
}

// match finds the handler and URL params for the given method+path.
// Returns (handler, params, allowedMethods, found).
// allowedMethods is non-empty when the path exists but the method is not registered (→ 405).
// Pre-allocates slices with capacity 4 (covers most real-world param counts without growing).
func (n *treeNode) match(method, path string) (HandlerFunc, *routeParams, string, bool) {
	keys := make([]string, 0, 4)
	vals := make([]string, 0, 4)

	h, allowed, found := n.matchInternal(method, path, &keys, &vals)
	if found {
		return h, &routeParams{keys: keys, vals: vals}, "", true
	}
	if allowed != "" {
		return nil, nil, allowed, false
	}
	return nil, nil, "", false
}

// matchInternal is the recursive worker for match.
// It appends matched params to *keys/*vals and backtracks on failure.
func (n *treeNode) matchInternal(method, path string, keys, vals *[]string) (HandlerFunc, string, bool) {
	if path != "" {
		// 1. Static children — try each child whose label matches path[0].
		label := path[0]
		for _, c := range n.children[ntStatic] {
			if c.label != label {
				continue
			}
			if !strings.HasPrefix(path, c.prefix) {
				continue
			}
			h, allowed, found := c.matchInternal(method, path[len(c.prefix):], keys, vals)
			if found || allowed != "" {
				return h, allowed, found
			}
		}

		// 2. Param child — at most one per level.
		if len(n.children[ntParam]) > 0 {
			c := n.children[ntParam][0]
			slash := strings.IndexByte(path, '/')
			var val, remaining string
			if slash < 0 {
				val, remaining = path, ""
			} else {
				val, remaining = path[:slash], path[slash:]
			}
			if val != "" {
				prev := len(*keys)
				*keys = append(*keys, c.paramKey)
				*vals = append(*vals, val)
				h, allowed, found := c.matchInternal(method, remaining, keys, vals)
				if found || allowed != "" {
					return h, allowed, found
				}
				// Backtrack.
				*keys = (*keys)[:prev]
				*vals = (*vals)[:prev]
			}
		}
	}

	// 3. Catchall child — captures any remaining path, including "".
	if len(n.children[ntCatchAll]) > 0 {
		c := n.children[ntCatchAll][0]
		prev := len(*keys)
		*keys = append(*keys, c.paramKey)
		*vals = append(*vals, path)

		if ep, ok := c.endpoints[method]; ok {
			return ep.handler, "", true
		}
		if len(c.endpoints) > 0 {
			return nil, allowedMethodsList(c.endpoints), false
		}
		// Backtrack.
		*keys = (*keys)[:prev]
		*vals = (*vals)[:prev]
	}

	// 4. Endpoint on the current node (exact match after prefix consumed).
	if ep, ok := n.endpoints[method]; ok {
		return ep.handler, "", true
	}
	if len(n.endpoints) > 0 {
		return nil, allowedMethodsList(n.endpoints), false
	}

	return nil, "", false
}

// longestCommonPrefix returns the length of the longest common prefix of a and b.
func longestCommonPrefix(a, b string) int {
	max := len(a)
	if len(b) < max {
		max = len(b)
	}
	for i := 0; i < max; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return max
}

// allowedMethodsList returns a sorted, comma-separated list of methods from an endpoints map.
func allowedMethodsList(endpoints map[string]*treeEndpoint) string {
	methods := make([]string, 0, len(endpoints))
	for m := range endpoints {
		methods = append(methods, m)
	}
	sort.Strings(methods)
	return strings.Join(methods, ", ")
}

// isStaticPattern reports whether a route pattern contains no URL parameters.
// Static patterns can be stored in a map for O(1) lookup.
func isStaticPattern(pattern string) bool {
	return !strings.ContainsAny(pattern, "{*")
}
