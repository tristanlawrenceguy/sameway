# chat

Use chat for a conversation: a named region holding a transcript and a
composer. It is an ordinary component, so on a Sameway canvas it is a block
like any other. It can be moved, widened, restyled with `layout`, or removed
entirely, and the assistant can put it back.

Because it can be removed, the conversation is never only here. The same
transcript and composer are always at `/chat`, and agents can post to
`/api/chat`, so nobody is locked out by a layout choice.

Use `layout: bare` when the region already sits inside a surface and the
panel chrome would double up. The title stays in the accessibility tree
either way.
