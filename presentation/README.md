# droidcon Uganda 2026 — Presentation

> Talk: **Building Civic Intelligence with Next.js + AI: Lessons from Kenya's Parliament**
> Speaker: Roy Wanyoike
> Format: Session, 30 minutes
> Date: October 28–29, 2026, Kampala

This folder contains everything needed to present the talk:

```
presentation/
├── README.md           ← this file
├── PROPOSAL.md         ← what gets submitted to droidcon (abstract + bio + takeaways)
├── slides.md           ← Markdown source for the 14-slide deck
├── slides.html         ← canonical, presentable reveal.js deck (open in a browser)
└── SPEAKER_NOTES.md    ← detailed talking points for each slide + Q&A prep
```

---

## How to open the slides

The deck is a single self-contained HTML file. No build step, no dependencies installed locally — everything loads from CDN.

**Quickest path:** double-click `slides.html`. It opens in any modern browser (Chrome, Firefox, Safari, Edge).

**Better path** (recommended for the live talk — avoids `file://` quirks with CDN fonts):

```bash
cd presentation
python3 -m http.server 8000
# then open http://localhost:8000/slides.html
```

Stop the server with `Ctrl-C` when done.

---

## Speaker notes

Speaker notes are embedded in two places:

1. **In the HTML deck** — every slide has an `<aside class="notes">` block. Open the speaker view by pressing **`S`** while the deck is open, or append `?notes` to the URL: `http://localhost:8000/slides.html?notes`. The speaker view opens in a separate popup window that stays in sync with the main deck.

2. **In `SPEAKER_NOTES.md`** — full prose notes for each slide, plus Q&A prep with likely questions and prepared answers, plus a timing table.

The HTML notes are condensed versions of the `.md` notes — use the Markdown file for your last-minute prep.

---

## Timing guide

| Section             | Slide(s) | Time   | Running total |
|---------------------|----------|--------|---------------|
| Title + intro       | 1        | 0:30   | 0:30          |
| The problem         | 2        | 2:00   | 2:30          |
| The opportunity     | 3        | 1:30   | 4:00          |
| Architecture        | 4        | 3:00   | 7:00          |
| Evidence-first AI   | 5        | 3:00   | 10:00         |
| Six reality labels  | 6        | 2:30   | 12:30         |
| Scenario engine     | 7        | 3:00   | 15:30         |
| Multi-country       | 8        | 2:30   | 18:00         |
| Uganda adapter      | 9        | 2:30   | 20:30         |
| Reality boundary    | 10       | 2:30   | 23:00         |
| Lessons learned     | 11       | 3:00   | 26:00         |
| Live demo           | 12       | 5:00   | overlap with slides 11/12 in practice |
| Open source         | 13       | 1:30   | 28:30         |
| Q&A                 | 14       | 4:00   | ~32:30        |

**Total:** ~30 min talk + ~2 min overrun buffer for Q&A. If running long, trim Q&A or compress the demo to 4 minutes (skip the GitHub repo walkthrough — point them to the link on the last slide instead).

**Practice tip:** if you find yourself running long on slide 5 (Evidence-first AI), cut the citation-validator bullet list — let the diagram do the work. The 5 checks are in the speaker notes if anyone asks.

---

## Demo checklist

The live demo runs from slide 12. **5 minutes, hard stop.** If you can't stay under 5 minutes, the demo is eating the talk. Cut it.

### Pre-demo checklist (do this BEFORE the talk)

- [ ] Frontend running locally on `http://localhost:3000` (or staging URL if WiFi is reliable)
- [ ] Go BFF running on `http://localhost:9000` — verify with `curl http://localhost:9000/api/v1/healthz`
- [ ] Python AI service running on `http://localhost:8000` — verify with `curl http://localhost:8000/healthz`
- [ ] At least one Bill pre-loaded into the BFF cache so the homepage shows real data immediately
- [ ] A pre-prepared "what if" scenario in `/scenarios` — don't create one live, the form takes 90 seconds
- [ ] Browser tabs open: `parliament.go.ug` (for the Uganda comparison), GitHub repo, terminal with `go test ./...` ready to run
- [ ] `OPENAI_API_KEY` env var set so the chat demo uses the real provider, not Stub
- [ ] WiFi checked — try loading the staging URL on conference WiFi the morning of the talk

### Demo flow (5 minutes, hard stop)

1. **Homepage (30s)** — search "What is happening with housing?" — show the results page. Don't read results aloud; let them appear.
2. **Bill detail (60s)** — pick the first Bill on the page. Show the plain-language summary, the verified timeline, the citation links. Click a citation — show it opens the original source document.
3. **"Ask about this Bill" chat (90s)** — ask "What stage is this Bill at?" Show the AI response with inline citations. Then ask a question the AI can't verify ("Will this Bill pass?") and show the `[This response could not be fully verified]` notice.
4. **Scenarios section (60s)** — open the pre-prepared "what if" scenario. Show the HYPOTHETICAL banner. Show the Monte Carlo output — median, p10, p90, min, max. Don't explain the math; point at the percentiles and say "a single number would be a lie."
5. **Uganda (60s)** — switch to the Uganda page. Show the Uganda Bill stages (8 stages, unicameral). Show the Uganda terminology ("Royal Assent — not applicable in Uganda"). Open `parliament.go.ug` in another tab to show the source.
6. **GitHub repo (30s)** — show the test count (456 Go tests, 22 Python tests). Show the adapter folder structure: `adapters/{kenya,uganda,tanzania,ghana,nigeria,south_africa}/`. Move to slide 13.

### Backup plan if demo fails

If WiFi dies, the BFF crashes, or anything else breaks:

1. **Don't apologize.** Say: "This is the part of the talk where I pretend the WiFi works. It doesn't. Let me show you screenshots instead."
2. The slides themselves contain the architecture diagrams and code snippets — you can talk through them without the live demo.
3. Have a few screenshots of the running platform pre-loaded in your photo viewer (homepage, Bill detail, chat with citations, scenario page, Uganda page) — about 5-6 images. Swipe through them.
4. If all else fails: skip the demo entirely, spend the recovered 5 minutes on slide 11 (lessons learned — you can extend it). End with the open-source slide and Q&A.

**The talk should still work without the demo.** The demo is the proof, not the content.

---

## Pre-talk rehearsal checklist

One week before:

- [ ] Run through the full talk with a stopwatch. Aim for 27 minutes — leaves 3 minutes of buffer.
- [ ] Practice the demo on conference-grade WiFi (e.g., a coffee shop, not your home network).
- [ ] Test the deck on the projector resolution you'll be using (most droidcon rooms are 1920×1080). If the projector is 4:3 instead of 16:9, the deck still works — reveal.js scales.
- [ ] Print the SPEAKER_NOTES.md (or have it on a tablet) for the live talk. Don't rely on the popup window — sometimes presenters' view doesn't work on stage displays.
- [ ] Check the GitHub repo is reachable from Uganda without a VPN. (It should be, but verify.)

Day of the talk:

- [ ] Arrive 30 minutes early. Test the projector with your laptop.
- [ ] Confirm `python3 -m http.server 8000` is running before you walk on stage.
- [ ] Open the deck, advance through every slide once to warm up the CDN caches.
- [ ] Run `go test ./...` in the repo — have the output ready if anyone asks "do the tests actually pass?"
- [ ] Close all other browser tabs. Notifications off. Do Not Disturb on.
- [ ] Water bottle on the podium. You'll talk for 30 minutes straight.

---

## If they reject the talk

The proposal is written to match droidcon Uganda's CFP guidelines explicitly (clear problem statement, concrete outcomes, real numbers, no buzzwords, intermediate level with case study, 30-min session). If it gets rejected, the most likely reasons:

- **"Not Android-specific enough."** Mitigation: the abstract and notes repeatedly call out how the patterns transfer to Android apps (provider-agnostic gateway, country-adapter pattern, reality-label visual language). If they push back, offer a 5-minute lightning version focused only on the adapter pattern.
- **"Too dense / too many topics."** The talk does cover a lot. Mitigation: drop slides 7 (scenario engine) and 10 (reality boundary) — keep them as backup. The talk still works at 22 minutes.
- **"Speaker not known."** True. The GitHub repo is the credential — 350+ files, 456 tests, 14 ADRs, 6 country adapters. The proposal mentions this directly.

If rejected, also submit to:

- **droidcon Kenya 2026** (the platform is Kenyan — even better fit)
- **DevFest Kampala 2026** (Google Developer Groups — broader audience, more tolerant of non-Android content)
- **FOSDEM 2027** (Open-source civic-tech devroom)

---

## License

The talk content (slides, proposal, speaker notes) is MIT — same as the platform itself. The reveal.js, highlight.js, and mermaid.js CDNs are their respective licenses (MIT). The Civic Intelligence color palette is from `apps/web/src/lib/design-tokens.ts`.
