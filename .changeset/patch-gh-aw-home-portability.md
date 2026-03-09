---
"gh-aw": patch
---

Allow gh-aw to derive runtime paths from the new `GH_AW_HOME` environment variable instead of enforcing `/opt/gh-aw`, so self-hosted runners can relocate the installation without recompilation.
