package dim

import (
	"sort"
	"strings"
)

// nodeTyp represents the type of a radix tree node.
type nodeTyp uint8

const (
	ntStatic   nodeTyp = iota // /users/
	ntParam                   // {id}
	ntCatchAll                // {path...}
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
	paramKey  string                   // key name for ntParam / ntCatchAll
	endpoints map[string]*treeEndpoint // method → endpoint
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
// allowedMethods is non-empty when the path exists but the method is not
// registered (→ 405). It is returned unsorted and possibly with duplicates so
// the caller can merge it with candidates of its own; joinAllowedMethods turns
// it into the header value.
// Pre-allocates slices with capacity 4 (covers most real-world param counts without growing).
func (n *treeNode) match(method, path string) (HandlerFunc, *routeParams, []string, bool) {
	keys := make([]string, 0, 4)
	vals := make([]string, 0, 4)
	var allowed []string // stays nil unless a 405 candidate is found

	h, found := n.matchInternal(method, path, &keys, &vals, &allowed)
	if found {
		return h, &routeParams{keys: keys, vals: vals}, nil, true
	}
	return nil, nil, allowed, false
}

// matchInternal is the recursive worker for match.
// It appends matched params to *keys/*vals and backtracks on failure.
//
// A branch whose path matches but lacks the requested method (405) does not stop
// the search: its methods are collected into *allowed and only reported when no
// branch at all produces a handler. Otherwise such a branch would mask a sibling
// branch that does handle the method.
func (n *treeNode) matchInternal(method, path string, keys, vals, allowed *[]string) (HandlerFunc, bool) {
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
			if h, found := c.matchInternal(method, path[len(c.prefix):], keys, vals, allowed); found {
				return h, true
			}
		}

		// 2. Param children — one segment each. Patterns differing only in the
		// param name land on separate children, so every child must be tried.
		if len(n.children[ntParam]) > 0 {
			val, remaining := path, ""
			if slash := strings.IndexByte(path, '/'); slash >= 0 {
				val, remaining = path[:slash], path[slash:]
			}
			if val != "" {
				for _, c := range n.children[ntParam] {
					prev := len(*keys)
					*keys = append(*keys, c.paramKey)
					*vals = append(*vals, val)
					if h, found := c.matchInternal(method, remaining, keys, vals, allowed); found {
						return h, true
					}
					// Backtrack.
					*keys = (*keys)[:prev]
					*vals = (*vals)[:prev]
				}
			}
		}
	}

	// 3. Catchall children — capture any remaining path, including "".
	for _, c := range n.children[ntCatchAll] {
		prev := len(*keys)
		*keys = append(*keys, c.paramKey)
		*vals = append(*vals, path)

		if ep, ok := c.endpoints[method]; ok {
			return ep.handler, true
		}
		// Backtrack.
		*keys = (*keys)[:prev]
		*vals = (*vals)[:prev]
		collectAllowedMethods(allowed, c.endpoints)
	}

	// 4. Endpoint on the current node — only once the whole path is consumed.
	// Without the path == "" guard the node would also answer deeper paths,
	// e.g. /m/{slug} serving /m/abc/x/y/z.
	if path == "" {
		if ep, ok := n.endpoints[method]; ok {
			return ep.handler, true
		}
		collectAllowedMethods(allowed, n.endpoints)
	}

	return nil, false
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

// collectAllowedMethods records the methods of a node whose path matched but
// whose method did not — the 405 candidates for the Allow header.
func collectAllowedMethods(allowed *[]string, endpoints map[string]*treeEndpoint) {
	for m := range endpoints {
		*allowed = append(*allowed, m)
	}
}

// joinAllowedMethods returns the sorted, de-duplicated, comma-separated Allow
// header value. Methods may come from several branches, so duplicates are
// possible. Mutates methods by sorting it.
func joinAllowedMethods(methods []string) string {
	sort.Strings(methods)
	uniq := make([]string, 0, len(methods))
	for i, m := range methods {
		if i > 0 && methods[i-1] == m {
			continue
		}
		uniq = append(uniq, m)
	}
	return strings.Join(uniq, ", ")
}

// isStaticPattern reports whether a route pattern contains no URL parameters.
// Static patterns can be stored in a map for O(1) lookup.
func isStaticPattern(pattern string) bool {
	return !strings.ContainsAny(pattern, "{*")
}
