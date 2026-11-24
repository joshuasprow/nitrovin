# The page's design

The page at `/` (`pkg/ui`) looks like a stamped VIN plate with tags hung from
it. [docs/design/page.md](docs/design/page.md) describes it as built, and this
file is that one's map. A change to the page follows that file, and updates it
in the same pull request. Detail goes there: this file stays within 1,600
bytes.

Each part has one of two standings. **Signature** parts are the page's
identity, and new UI is built from them. **Good enough** parts stay as they
are: a change fixes a bug or follows a signature part.

## Rules

- [Ground rules](docs/design/page.md#ground-rules): what the page is built
  with.
- [Composition rules](docs/design/page.md#composition-rules): how a change is
  put together, each from a correction.
- [Building something new](docs/design/page.md#building-something-new).
- [Widths](docs/design/page.md#widths): where the page changes, and why.

## Colors

[Tokens](docs/design/page.md#tokens): custom properties on `body` in
[ui.css](pkg/ui/static/ui.css). The ground, the text, the dialogs and the six
group colours change between light and dark. The four materials do not.

## Typography

[Type roles](docs/design/page.md#type-roles): a label face for what is
stamped, a monospace for data, `system-ui` for prose.

## Components

Each is a section of that name.

- **Signature:** the VIN input, the plate's second mode, aluminum plates, vinyl
  tape, manila tags, thermal paper.
- **Good enough:** what a card holds, the parts that fit, empty places, where
  the brackets and lines run, dialogs, forms and inputs.
