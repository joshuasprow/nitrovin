# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

Two audiences, in this order.

- **People in the parts and salvage trade.** Yard staff, and a catalog's
  author or receiver (ADR 0023). They have one vehicle in front of them and
  want its decode, its names in their own vocabularies, the part types that
  fit it, and what is in stock. They value density and speed over looks
  ([docs/research/vin-page.md](docs/research/vin-page.md)).
- **Developers evaluating nitrovin.** They decide whether to wire up `/v1`,
  the CLI, stdio, MCP, the Go module or the C library. For them the page is
  a live demonstration of the contract.

## Product Purpose

nitrovin is an offline VIN decoder that answers like NHTSA's vPIC. It is a Go
port of `spVinDecode` and ships as one binary with one data month of vPIC
inside. Around decode it translates through vocabulary files, fits against a
fitment catalog, and looks up stock.

Its goals are compatibility, answering exactly as vPIC does, and
interoperability, working with other systems through vPIC's ids, its output
shapes and standard formats. Speed serves those goals and is not one of them.

Success is a data month that ships only when it matches the oracle, and an
answer that is the same on every surface.

## Positioning

- **It answers exactly as vPIC does.** Every row of every VIN in the parity
  corpus matches NHTSA's own procedure byte for byte, or the month does not
  ship ([docs/parity.md](docs/parity.md)).
- **It works offline.** One binary; no network, no database, no API key.
- **A visitor's files stay in the browser.** Vocabularies and catalogs load
  into the browser and translate there. They are never uploaded (ADR 0013,
  ADR 0023).
- **One contract, six surfaces.** `/v1` is frozen and additive only. The CLI,
  HTTP, stdio, MCP, the Go module and the C library answer through the same
  `Translator`.
- **It is vendor-neutral.** Public text names no deployer's vendors, products
  or data. nitrovin is not affiliated with or endorsed by NHTSA.

## Operating Context

- [nitrovin.app](https://nitrovin.app) runs the latest release's server for
  anyone, rate-limited per IP. It ships no catalog.
- A deployer runs `nitrovin serve` with its own vocabularies, catalog and
  stock connector.
- The browser surface is one page, `GET /` (`pkg/ui`): the VIN's diagram
  (ADR 0027). It takes a VIN and nothing else.
- A visitor loads their own vocabulary files, EPA's vocabulary, or the demo
  set, each in one click.
- Fitment drills down in place on that page, and the open state is in the URL.

## Capabilities and Constraints

- **Requests:** decode, translate, list, fit and stocked.
- **A bad VIN is not an error.** Partial and mistyped VINs decode, and their
  error codes go in the decode.
- **"Nothing to fit" is a status,** not a refusal. Vocabulary drift is
  reported; faults refuse loading.
- **Every refusal is an RFC 9457 Problem.**
- **The page works without JavaScript.** Its CSP refuses inline style and
  inline script, and it uses no raster images
  ([docs/design/page.md](docs/design/page.md#ground-rules), "Ground rules").
- **The data month comes from the `nitrovin-data` pin in `go.mod`.** No text
  hard-codes it as current.
- **Words follow [GLOSSARY.md](GLOSSARY.md),** never an entry's `_Avoid_`
  synonyms.
- **Status:** v0.x prereleases. The contract is frozen; v1.0.0 waits on
  ADR 0007's gates.

## Brand Commitments

- **Name:** nitrovin, one name for every surface (ADR 0002).
- **Voice:** declarative, present tense, third person. It states counts and
  roles, never superlatives.
- **Visual identity:** [DESIGN.md](DESIGN.md) is the authority. Its signature
  parts are the VIN input and the four materials.
- **Credit:** NHTSA's vPIC is the source. ultravin is a reference, credited
  in `NOTICE`, and never copied.

## Evidence on Hand

- The parity gate and its corpus: [docs/parity.md](docs/parity.md),
  `testdata/parity/<month>/`.
- Registered deviation classes, each with a live-vPIC witness:
  [docs/deviations/README.md](docs/deviations/README.md).
- Measured benchmarks: [docs/benchmarks.md](docs/benchmarks.md). They gate
  nothing.
- The demo set and the synthetic fixture in `testdata/vocab/`. These are the
  only data a public surface shows.
- Signed releases, verifiable with cosign.
- **Absent, and never to be invented:** testimonials, named customers or
  deployers, pricing, and any endorsement by NHTSA.

## Product Principles

1. **Parity before everything.** A difference from vPIC is a bug or a
   registered deviation class, never a per-VIN excuse.
2. **Say what is known, as vPIC says it.** Doubt is written where it applies:
   a status, a drift finding, a caveat sentence.
3. **The same answer everywhere.** A new capability reaches every surface
   through the one contract.
4. **The visitor's data is the visitor's.** Files are read where they are and
   never uploaded.
5. **Density and speed serve the parts trade.** The person at the page is
   mid-task with a vehicle in front of them.

## Accessibility & Inclusion

WCAG 2.2 AA. The page also keeps what [DESIGN.md](DESIGN.md) already
requires: it works without JavaScript, it shows no motion under
`prefers-reduced-motion`, and the VIN input stays one native field, so paste,
selection and the phone's keyboard keep working.
