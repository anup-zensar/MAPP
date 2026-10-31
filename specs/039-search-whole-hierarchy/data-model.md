# Data model: Search the whole hierarchy

No table changes. Two answer shapes (contract in [contracts/api.md](contracts/api.md)):

- **PlanSearchMatch**: level, ID, name, parent's ID of one record below a plan whose name or ID contains the text.
- **FoundIn**: on a Plans list item, the first match top-down and the number of further matches.

Records in the recycle bin are not stored in the level tables, so they never match (011).
