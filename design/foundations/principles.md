# Principles

1. **Same way for everyone.** A screen reader, a keyboard, a pointer, and an
   agent driving a browser all navigate by the same landmarks, roles, names,
   and attributes. Nothing is built twice.

2. **The model designs, the person writes.** Layout, tone, which component,
   how wide, what order: that is the assistant's, because describing it in
   words is faster than any set of controls. A person edits the words in
   place and takes actions. Everything else they ask for. This is why the
   interface has almost no chrome: most of it would be controls for
   decisions nobody makes by hand.

3. **Say it three times.** Every fact that matters is available as text a
   person reads, as a visual a sighted person scans, and as an attribute a
   machine queries. Colour, motion, and position never carry meaning alone.

4. **Show who and what changed.** People and the assistant act on the same
   canvas. Every block says who added it and who last touched it; every turn
   leaves a receipt; every action lands in the activity log.

5. **Motion explains, never decorates.** Things animate when they enter,
   leave, or change, so the eye can follow. Nothing moves on its own, nothing
   loops, and it all stops under reduced motion with the meaning kept.

6. **Enforced, not aspirational.** Contrast, keyboard operation, golden
   output, and manifest completeness are tests that fail the build. If it is
   not tested, it is not a rule.
