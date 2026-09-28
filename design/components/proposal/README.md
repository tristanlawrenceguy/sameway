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

## Why it works this way

- **The model asks only about the canvas.** A question in its own words may
  carry adding, changing or removing a block or a tab, all of which can be
  undone. A setting, a command, letting someone in or deleting a field is
  asked by Sameway in words the code writes, so a Yes never agrees to
  something other than what the question said
  ([OWASP, excessive agency](https://genai.owasp.org/llmrisk/llm062025-excessive-agency/)).
- **Answered once.** Yes and No pressed together, or one sent twice, is
  answered once; the buttons wait, still in reach, while it is sent
  ([Adrian Roselli on disabled controls](https://adrianroselli.com/2024/02/dont-disable-form-controls.html)).
- **What cannot be undone says so**, in words and with a mark, and its Yes
  is in the danger colour, not the same filled button as a harmless one
  ([GOV.UK warning button](https://design-system.service.gov.uk/components/button/)).
- **The question names the group and what would happen describes it**, so
  someone moving to the buttons hears both
  ([APG names and descriptions](https://www.w3.org/WAI/ARIA/apg/practices/names-and-descriptions/)).
- **Answers say what they do** ("Send it", "Hide it instead"), and after
  answering the page says what was done, "Done: Hide it instead."
  ([NN/g on confirmation dialogs](https://www.nngroup.com/articles/confirmation-dialog/)).

Not done, and why: typing a word to confirm (a real burden for people with
cognitive disabilities, and the question is already plain); a modal dialog
(it needs scripts and breaks the conversation's flow).
