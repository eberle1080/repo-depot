This is repo-depot. This is an empty repository right now, and even this CLAUDE.md
will become out of date, almost as soon as the first commit drops.

Here's why this exists (a copy-paste from a chat log):

I want to write a small program which will manage local repositories for AI agents. Something I can expose via CLI. It will keep a centralized bare git clone somewhere canonical (on disk) and then instead of doing the git clone, git checkout stuff - the agent would register a workspace (on disk) and then invoke the cli to work on various repos within that workspace. Internally this would do a git clone against the bare repo (I want clone because actual git workspaces have too many restrictions) and then set up that clone's remote to point at the real upstream repo (github). Essentially this is kind of a git-clone-accelerator, but also it keeps track of these workspaces. Two questions: 1. Does this solution already exist somewhere? Seems like a git proxy almost, I don't want to re-invent the wheel. But the workspace tracking is unique, so maybe less common. 2. What should I call this? I need a name. I like fun, clever names.

The answer to 1 is, yeah kinda but not really, and the answer to 2 is repo-depot. And now dear reader you're all caught up.
