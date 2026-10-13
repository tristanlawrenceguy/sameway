# On your other computers

One workspace on a laptop and a desktop, or on a partner's computer, kept
in step through the cloud folder they already have (OneDrive, Dropbox,
iCloud Drive, Google Drive). Each computer keeps a whole copy and works
without the other; changes pass between them through the folder. This
is where the rules are, and why. The code is `internal/cloudsync`,
`internal/server/cloud_sync*.go`, and the stamps in
`internal/store/state.go`, `clash.go` and `setting_state.go`.

## What the owner sees, on Workspaces

Under **On your other computers**, one sentence says where it stands:

- In step: how many other computers have it open, that a change reaches
  them once the cloud app has copied it, and **Last in step with
  OneDrive today at 2:05pm**.
- Stopped: what stops it, and **What you change here is kept, and passes
  on once** that is put right. Its folder gone from the cloud folder
  offers **Put its folder back in OneDrive**.

The same stop is the job's error under Work in the background on Help,
and in `GET /api/jobs` for an agent.

## Why it works this way

- **Every computer is whole on its own.** Each copy is the primary copy,
  works offline and passes changes when it can: the local-first ideals
  of no spinners, many devices, and the network optional
  ([Ink & Switch](https://www.inkandswitch.com/essay/local-first/)).
- **A change is never lost.** A change is a stamp kept in the database,
  written to the folder as a whole batch that is never changed after; a
  batch half arrived is read next time, and one read twice changes
  nothing. Two people changing different fields of one record both keep
  theirs. Two changing the same text at once keep both: the one that
  lost is offered back on the record's page, as Dropbox's conflicted
  copy and Obsidian's conflict file keep both versions, but in the
  record rather than as a stray file
  ([Dropbox](https://help.dropbox.com/organize/conflicted-copy),
  [Obsidian](https://obsidian.md/help/sync/troubleshoot)). Two changing
  one short field (a date, a choice) at once agree on the later: Ink &
  Switch found such clashes rare, and either outcome acceptable.
- **A folder that goes is never read as things deleted, and is not put
  back on its own.** Deletion is a stamp, so a missing folder deletes
  nothing. But the folder may have gone on purpose, so keeping in step
  stops and says so until the owner puts it back, as Syncthing stops
  when its folder marker is missing
  ([Syncthing FAQ](https://docs.syncthing.net/users/faq.html)). Put back
  from one computer, the others pick up on their own, and each writes
  all its changes again, so one written just before the folder went and
  not yet read still arrives.
- **Never silence.** A cloud app signed out used to stop keeping in step
  without a word, and a folder gone showed the program's own error. The state, and when it was last
  in step, is said where the owner turned it on: people trust what keeps
  them told ([NN/g, visibility of system
  status](https://www.nngroup.com/articles/visibility-system-status/)).
  "Last in step with" is the moment this copy had read everything in the
  folder and written everything of its own; the other computers follow
  as their cloud app copies, which no app can promise a time for.
- **The time zone is the person's, so it is the same everywhere.** A
  workspace keeps one zone (today.md); set on the laptop, the desktop's
  days follow it, and the later choice wins on both. It travels as a
  stamp, like a record. Only settings that are the person's travel, and
  only `ui.zone` so far: a copy never takes another's programs to run,
  folders or model, whatever it is sent.
- **Clocks that disagree do not reorder changes.** Stamps carry a hybrid
  logical clock: real time, moved past any stamp heard of, so a change
  made after reading another always sorts after it, whatever the two
  computers' clocks say
  ([Kulkarni et al.](https://cse.buffalo.edu/tech-reports/2014-04.pdf)).
  It stays on real time while the app's clock may be a test's (the
  store reads the app's clock for everything else).
- **Agents.** An agent reads the stop from `GET /api/jobs`, and the
  same sentence on Workspaces. Turning it on, off or back is the
  owner's own, by a person's press: which computers hold a copy is
  theirs to choose.

## Not done, and why

- **Refusing or flagging a computer whose clock is far ahead.** Such a
  computer wins a clash it should not, but refusing its changes would
  lose them, and clashes on text keep both versions anyway. Computers
  set their clocks themselves.
- **Other settings travelling (the morning brief's time, the language,
  the clock face).** Each is arguably the person's too, but whether a
  laptop and a desktop should share them is not clear; the zone is the
  one that makes the days disagree.
- **A live "syncing now" indicator.** A round takes a moment every
  fifteen seconds; a spinner would say nothing the sentence does not.
- **Merging two versions of a text into one, as Obsidian does by
  default.** Its forum reports merges that read wrong; keeping both and
  offering the other back loses nothing and guesses nothing.
