# Home visual baseline

- Viewport: 375×812.
- First-viewport order: city, search, current order, nearby sites, site cards.
- Background: `#F7F9FC`; surfaces: pure `#FFFFFF`; primary: `#1769E0`.
- No greeting hero, skyline, map, marketing badge, cream background, or shipping workflow.
- Current-order empty state is shown until plan 2 connects the order API; no fake active order appears in production runtime.
- Site names, distance, availability, status, and long Chinese address remain readable without horizontal overflow.
- Loading uses neutral skeleton blocks; empty, network, and offline states state the impact and next action.
- Site detail counts only cells whose status is `CELL_STATUS_IDLE`.
