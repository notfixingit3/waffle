# Instagram In-App Browser Findings Template

## Severity Taxonomy

Use these definitions when classifying every finding. If a finding could fit two levels, choose the higher one.

- **Critical** — The claim flow or an admin flow is broken and cannot be completed. Example: tapping Claim does nothing, login loops, mark-paid modal never opens, or WebSocket updates never arrive across devices.
- **High** — A major UX defect with a workaround. Example: the user must tap the Claim button twice, must close and reopen the page to see updates, or must refresh after every navigation.
- **Medium** — Cosmetic or intermittent. Example: a brief flash of the offline banner, the live indicator flickers between "connecting" and "live", or the grid momentarily overlaps while scrolling.
- **Low** — Polish. Example: a slightly off-center button, color mismatch, or text that could be clearer. These do not block usage.

## Per-Finding Fields

Record each finding as a single row in the table below. Every field is required. If a field genuinely does not apply, write "N/A" rather than leaving it blank.

- **Date** — The date you found it.
- **Platform** — `iOS` or `Android`.
- **Track** — `Public iOS`, `Public Android`, `Admin iOS`, `Admin Android`, or `Cross-platform`.
- **Severity** — Critical, High, Medium, or Low.
- **Repro Steps** — The exact steps from the checklist that led to the issue. Be specific enough that another person can repeat it.
- **Expected** — What the checklist step says should happen.
- **Actual** — What actually happened, including exact wording of any error message.
- **Evidence / Screenshot** — File name or link to a screenshot, screen recording, or log snippet. Store these in the same folder or a shared drive and reference them here.
- **Suspected Cause** — Your best guess based on what you saw. Keep it short and honest. Examples: "iOS keyboard-close viewport shift", "localStorage wiped by IG WebView", "Meta pcm.js interference", "memory-pressure WebSocket drop not reconnecting".
- **Environment** — `http://<LAN-IP>:8383` or `https://dev.waffle.projectsyrup.app`.
- **Browser / App Version** — Instagram app version and OS version if you know them. Example: "Instagram 428.2.0 on iOS 26.5".

## Findings Table

| Date | Platform | Track | Severity | Repro Steps | Expected | Actual | Evidence / Screenshot | Suspected Cause | Environment | Browser / App Version |
|------|----------|-------|----------|-------------|----------|--------|-----------------------|-----------------|-------------|-----------------------|
| | | | | | | | | | | |
| | | | | | | | | | | |
| | | | | | | | | | | |
| | | | | | | | | | | |

## Report-First Note

This document is a findings report, not a work order. Every finding above becomes a follow-up plan if the team decides to fix it. Do not hotfix code or patch the app during this pass. The goal is to know what Instagram's browser is doing, not to fix it on the spot. When this checklist is complete, hand the findings table to the team for triage and planning.
