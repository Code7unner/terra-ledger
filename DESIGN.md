# Design System -- TerraLedger

## Product Context
- **What this is:** Agricultural Credit Intelligence platform on Solana for Kazakhstan
- **Who it's for:** Farmers (register land, get credit score) and Lenders (search parcels, assess risk)
- **Space/industry:** Agtech + Fintech + Blockchain (RWA tokenization)
- **Project type:** Web app with marketing landing page

## Aesthetic Direction
- **Direction:** Industrial/Utilitarian with Fintech trust signals
- **Decoration level:** Intentional (subtle glow effects, grid patterns, not excessive)
- **Mood:** Dark, technical, trustworthy. Like a Bloomberg terminal meets a satellite control room. Data-dense but readable. The user should feel "this is real infrastructure, not a toy."
- **Reference sites:** Marinade Finance (pulsing CTAs, dark teal), Solana.com (stat grids), Regrow Ag (pipeline tabs), Planet Labs (satellite imagery)

## Typography
- **Display/Hero:** Space Grotesk (700) -- geometric, technical, uppercase for impact
- **Body:** Inter (400/500) -- clean workhorse, high legibility at small sizes
- **UI/Labels:** JetBrains Mono -- signals technical credibility, used for stats, addresses, badges
- **Data/Tables:** JetBrains Mono (tabular-nums) -- fixed-width digits for aligned numbers
- **Code:** JetBrains Mono
- **Loading:** Google Fonts (`Space+Grotesk:wght@600;700&family=Inter:wght@400;500;600&family=JetBrains+Mono:wght@400;500`)
- **Scale:** caption(12px) body-sm(14px) body(16px) h3(20px) h2(24px) h1(32px) hero(56px, 44px tablet, 32px mobile)

## Color
- **Approach:** Restrained (one accent + neutrals, color is rare and meaningful)
- **Primary:** #25d0ab -- teal/emerald, represents growth + trust + agriculture
- **Primary hover:** #1fa88a
- **Primary bg:** #04312c (muted dark teal for badges/eyebrows)
- **Background:** #0f0f0f (near-black)
- **Surface:** #141414 (cards, panels)
- **Surface hover:** #1c1c1c
- **Text:** #ededed (primary text)
- **Text secondary:** #a0a0a0 (labels, descriptions)
- **Text muted:** #505050 (hints, timestamps)
- **Border:** #232323
- **Border hover:** #343434
- **Semantic:** success #25d0ab, warning #f59e0b, error #ef4444, info #25d0ab
- **Dark mode:** This IS dark mode. No light mode.

## Spacing
- **Base unit:** 4px
- **Density:** Comfortable
- **Scale:** xs(4) sm(8) md(16) lg(24) xl(32) 2xl(48) 3xl(64)

## Layout
- **Approach:** Grid-disciplined (strict columns, predictable alignment)
- **Grid:** 3-column for stats/pipeline, 2-column for role cards, 1-column mobile
- **Max content width:** 960px (app), full-bleed with 960px inner for landing
- **Border radius:** 4px everywhere (sharp Colosseum style). Full (9999px) for pills/badges only.

## Motion
- **Approach:** Intentional (scroll-triggered fade-ins, counter animations, pulsing CTA)
- **Easing:** enter(ease-out) exit(ease-in) move(cubic-bezier(0.4, 0, 0.2, 1))
- **Duration:** micro(100ms) short(200ms) medium(400ms) long(600ms)
- **Scroll animations:** Elements start at opacity:0 translateY(16px), transition to visible on IntersectionObserver trigger (threshold 0.1)
- **Counter animation:** 1500ms ease-out cubic (1 - Math.pow(1 - t, 3)) via requestAnimationFrame
- **CTA pulse:** 2.5s infinite box-shadow pulse in primary-ring color
- **Scroll indicator:** Mouse icon with 2s wheel animation loop, fades on scroll

## Landing Page Patterns
- **Hero:** Full-viewport, centered content, radial gradient glow (rgba(37,208,171,0.05)), entrance animation 600ms
- **Stats:** 3-column grid with animated counters, mono font, primary color numbers
- **Pipeline:** 3-step cards with numbered badges, dashed connecting line
- **Role cards:** 2-column with hover lift (-2px), border-color transition to primary
- **External links:** Pill-shaped, mono font, uppercase
- **Scroll indicator:** CSS-only mouse icon with teal wheel dot animation

## Decisions Log
| Date | Decision | Rationale |
|------|----------|-----------|
| 2026-04-06 | Initial design system | Codified from existing tokens.css + competitive research (Marinade, Solana, Regrow, Planet Labs) |
| 2026-04-06 | Keep sharp 4px radius | Colosseum hackathon identity, consistent with Solana ecosystem |
| 2026-04-06 | No light mode | Hackathon demo is dark-first, matches Colosseum + Solana brand |
| 2026-04-06 | Pulsing CTA pattern | Borrowed from Marinade Finance, subtle but draws eye to primary action |
