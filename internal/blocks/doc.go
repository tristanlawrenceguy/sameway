// Package blocks is the page model without HTTP: what a block shows,
// worked out from its props, the workspace's records and where it is
// shown. A data-bound component (a chart, a tracker, the clock) is a
// Kind here, which resolves its props to what its template draws and
// says in a few words what it shows. The pages draw what it resolves;
// the assistant and an agent over MCP ask it what a block will show
// before writing one, and get the same answer the page would give,
// without a web server.
//
// It sits above records, store and schema, and below server and chat.
package blocks
