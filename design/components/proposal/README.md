# proposal

Use a proposal when the assistant wants to do something it should ask about
first: anything that removes something, and anything it is guessing at. The
question and both answers sit together, so nobody agrees to a surprise.

Say what happens in the button, never "OK". Add `detail` naming exactly what
would change. Nothing is preselected, nothing is on a timer, and nothing
happens until a person picks one.

The assistant creates these with the `propose_change` tool. Accepting runs
the change through the same tools the assistant would have used itself, so
there is no second code path that could behave differently.
