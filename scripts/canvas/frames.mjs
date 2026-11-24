// The six frames of the design canvas (docs/design/canvas): what each one
// shows, as specimens of the page. A specimen names a width class (vp), a
// URL, the elements to keep (their ancestors stay, their siblings go), and
// any popover to draw open. Every URL is the demo set's.
import { BARE } from './engine.mjs';
const F150 = '1FTEW1EG7FFA12345', ACCORD = '1HGCM82633A004352', F13 = '1FTFW1EF6DFA12345', CAMRY = '4T1B11HK7JU512345', SILV = '1GCVKREC6EZ123456', RAM = '1D7RV1GT5AS123456';
const W = 1280;
const vin = (v, q = '') => '/?from.vin=' + v + q;
const PLATE = ['#lookup', '#answer > .under'];
const blur = async (c) => { await c.eval('document.activeElement && document.activeElement.blur(); 1'); await c.sleep(100); };
const type = (text, then) => async (c) => {
  await c.eval('document.querySelector("#from-vin").focus(); 1');
  await c.send('Input.insertText', { text }); await c.sleep(900);
  if (then) await c.eval(then + '; 1'); await c.sleep(200);
};
const showVocab = async (c) => { await c.eval('document.querySelector("#vocabularies").showPopover(); 1'); await c.sleep(4000); };
const DIAGRAM = ['#lookup .plate-form', '#answer > .under', '#answer > .leaders', '#answer > .hang', '#fit'];
// The picker's list is drawn where the page hangs it: from the picker, at
// its width (the page's own rule), 30rem tall at most and scrolling. The
// engine draws every other open popover in the flow and whole.
const LIST = '.sheet .vp #fit .ptdialog.is-open{top:calc(anchor(bottom) + var(--s1)) !important;left:anchor(left) !important;max-height:30rem !important;overflow:auto !important}';

export const frames = [
  {
    file: 'Main', title: 'The VIN input, both modes', w: W,
    heading: 'The VIN input, both modes',
    intro: 'The form is the plate: 17 struck cells under six group labels, drawn over one real text field. A switch above it turns the plate to its second mode, a part type and a part number, on the same metal.',
    items: [
      { vp: 'wide', url: '/', keep: PLATE, pre: blur, cap: 'Empty', note: 'The placeholder is stamped across the plate in the label face. The switch is two keys; the pressed one is the mode shown.' },
      { vp: 'wide', url: '/', keep: PLATE, pre: type('1FTEW1EG'), cap: 'Typing, focused', note: 'The next cell blinks a bar in the link colour. A finished group\'s bracket becomes pinstripe tape in its colour.' },
      { vp: 'wide', url: vin(F150), keep: PLATE, pre: async (c) => { await c.eval('const t = document.querySelector("#from-vin"); t.focus(); t.setSelectionRange(3, 8); document.dispatchEvent(new Event("selectionchange")); 1'); await c.sleep(300); }, cap: 'Selection', note: 'A selected cell is inked dark, ringed in its group\'s colour.' },
      { vp: 'wide', url: '/', keep: PLATE, pre: async (c) => { await type('1FTEW1EG7FFO12345')(c); await blur(c); }, cap: 'A character no VIN holds', note: 'I, O and Q fill their cell with the error colour; the plate still decodes.' },
      { vp: 'wide', url: '/?from.fitment.number=', keep: ['#lookup'], pre: blur, cap: 'Part-number mode, empty', note: 'The second mode: a part type key and a number field, each under its label and bracket.' },
      { vp: 'wide', url: '/?from.fitment.type=alternator&from.fitment.number=DEMO-1001', keep: ['#lookup'], pre: blur, cap: 'Part-number mode, filled', note: 'The part type chosen and the number typed.' },
      { row: [
        { vp: 'phone', url: vin(F150), keep: PLATE, pre: blur, cap: 'Phone, VIN', note: 'Two rows of cells.' },
        { vp: 'phone', url: '/?from.fitment.type=alternator&from.fitment.number=DEMO-1001', keep: ['#lookup'], pre: blur, cap: 'Phone, part number', note: 'The fields stack.' },
      ] },
    ],
  },
  {
    file: 'Materials', title: 'Materials and elevation', w: W,
    heading: 'Materials and elevation',
    intro: 'Four materials, the same in both colour schemes: brushed aluminum, vinyl tape, manila and thermal paper. Three heights: static tape is stuck on; a control stands raised on a steel ledge; pressed, it sits flush, 2px down and darker.',
    items: [
      { h2: 'Aluminum' },
      { vp: 'wide', url: vin(F150), keep: ['header', '#lookup', 'footer'], pre: blur, mark: { '.open-vocabularies': 'is-active' }, cap: 'Plates and keys', note: 'The brand is a static plate. A button is a key of the same metal, raised; Vocabularies is shown pressed, as is the VIN key of the switch. VINs ends the row over the plate, and Developers and the release are the foot.' },
      { h2: 'Vinyl tape' },
      { vp: 'wide', url: vin(F150, '&part.id=alternator'), keep: ['#fit .fitstrip', '#fit .ptwrap', '#fit .aka', '#fit .cands'], cap: 'Label tape, the second tape, pinstripe', note: 'Black label tape carries the vehicle\'s block, a vocabulary\'s name on its tag and a candidate\'s number, letters raised white. The second tape is the plate\'s metal with the tape\'s ridges: the part type picker is a key of it on its own piece of label tape, and a candidate\'s stock strip is the trailing end of its tag. Pinstripe tape is a group\'s colour, lit on one edge: the cards\' lines end on the block\'s top edge, and one trunk runs down from the block, a stub to the picker and to each candidate.' },
      { h2: 'Manila' },
      { vp: 'wide', url: vin(F150, '&part.id=alternator'), keep: ['#answer > .leaders', '#answer > .hang > .cards', '#fit .aka', '#fit .cands'], cap: 'Tags', note: 'Paper stock with tooth and a reinforced eyelet. A card hangs upright, its top corners clipped, a band printed in its group\'s colour. Under the cards a tag is laid long, its eyelet at its leading end and its heading on label tape: a vocabulary\'s names, and a candidate\'s row. A control on a tag is label tape; a candidate\'s stock strip is the one metal on a tag.' },
      { h2: 'Thermal paper' },
      { vp: 'wide', url: vin(F150, '&part.id=alternator&fitment.number=DEMO-1001'), keep: ['#fit .cands'], cap: 'A print', note: 'What a stock file says of a part: a strip of paper fed out from under its tag, torn at the foot, in one near-black in the monospace face. A reversed bar names each kind of stock, its source\'s date at the right. A row is its id, its location in bold, then a column for each field its source\'s header declares, under the field\'s name. A field the header leaves out is folded under its row in the fainter print, here a yard row\'s colour, and a note is a starred line.' },
    ],
  },
  {
    file: 'Diagram', title: 'Cards, brackets and lines', w: W,
    heading: 'Cards, brackets and lines',
    intro: 'Each card is a manila tag hung by its eyelet from its group\'s line. A row is a label over its value, both left-aligned. Under the cards, the vehicle\'s block of label tape says what the vehicle is, and the Maker, Vehicle and Model year lines end on it. One trunk runs down from the block to the part type picker, and the vehicle\'s names hang from the block beside the picker as manila tags, a vocabulary each.',
    items: [
      { vp: 'wide', url: vin(F150), keep: DIAGRAM, pre: blur, cap: 'Decoded', note: 'Brackets, leader lines, cards, the lines to the block, the block, the trunk to the picker and the names. Nothing is pressed, so a blank strip asks for a part type.' },
      { vp: 'wide', url: vin('1FTEW'), keep: DIAGRAM, pre: blur, cap: 'Waiting', note: 'A group not yet typed keeps its place: a faint bracket and line, and a dashed outline where its tag will hang. No block is drawn, and no line leaves the cards.' },
      { row: [
        { vp: 'tab', url: vin(F150), keep: DIAGRAM, pre: blur, cap: 'Narrow', note: 'Below 64rem the cards are one sideways-scrolling row, which fades at an end with more beyond it, with no leader lines. Check, Year and Plant, each wider than its cell, stand clear of one another. The block\'s words are packed at its left, and the lines to them are carried from the row. The picker keeps a wide page\'s width, and the names\' row scrolls by itself beside it.' },
        { vp: 'phone', url: vin(F150), keep: DIAGRAM, pre: blur, cap: 'Phone', note: 'The same row at 390, under a plate of two rows. The picker is the row\'s width. The names are the last row on the trunk, here under the blank strip that asks for a part type.' },
      ] },
    ],
  },
  {
    file: 'Fitment', title: 'The fitment card', w: W, css: LIST,
    heading: 'The fitment card',
    intro: 'The parts that fit. Under the vehicle\'s block the page is a counter: a picker of the catalog\'s part types with the vehicle\'s names beside it, then the candidates as rows down the page, each a manila tag laid long and tied by a stub to the one trunk that runs down from the block. A candidate\'s stock strip feeds out its shelf and yard rows, a print in columns on thermal paper.',
    items: [
      { vp: 'wide', url: vin(F150, '&part.id=alternator&fitment.number=DEMO-1001'), keep: ['#fit'], cap: 'Picker, candidates and stock', note: 'Alternator pressed: the picker names it and counts its candidates, and × beside it clears. On the block, what the decode says of this vehicle, to match each candidate\'s options against. A candidate is a row: its number on label tape with its years, its option text over its mark and its note, and its stock strip at its trailing end. The first candidate\'s stock is open under its row. The shelf\'s header declares the stock number and the condition, and the yard\'s the stock number and the row; the colour, which it leaves out, is folded.' },
      { vp: 'wide', url: vin(F150, '&part.id=alternator'), open: ['#part-types'], keep: ['#fit'], style: 'padding-bottom: 340px', cap: 'The picker\'s list', note: 'Hung from the picker at its width, downward, over the candidates: a filter, the part types opened recently, then every part type under the part vocabulary\'s headings, in an order that never changes. The pressed one is flush; pressing it again changes nothing. Past 30rem it scrolls.' },
      { vp: 'wide', url: vin(SILV, '&part.id=brake-rotor-rear'), keep: ['#fit'], cap: 'Marks', note: 'One row. A solid stamp says what was checked against the decode, and a second names the part type the catalog lists the part under.' },
      { row: [
        { vp: 'tab', url: vin(F150, '&part.id=alternator'), keep: ['#fit'], cap: 'Narrow', note: 'Below 64rem a row is 3.5rem, a finger\'s. The picker keeps a wide page\'s width. A number\'s years are under it, and a stock strip is a column, no narrower than 7.5rem. The names scroll beside the picker.' },
        { vp: 'phone', url: vin(F150, '&part.id=alternator&fitment.number=DEMO-1001'), keep: ['#fit'], cap: 'Phone', note: 'A row is two lines: its number and its stock strip, a row high, then its ink. The print keeps its columns and scrolls sideways under its bar, its location staying in sight, and what is folded keeps the paper\'s width. The names are the last row on the trunk, under the candidates.' },
      ] },
      { vp: 'wide', url: '/?from.fitment.type=alternator&from.fitment.number=DEMO-4001&fitment.number=DEMO-4001', keep: ['#answer'], cap: 'Part-number mode', note: 'The same rows. The number\'s line runs into the eyelet of the part\'s tag, a row as wide as the plate, its print open under it. The trunk leaves the tag in the Serial group\'s colour and runs down through the vehicles the part fits, each a row: the catalog\'s name and years on label tape, then its option text over its names in the vocabularies loaded, underlined ink.' },
    ],
  },
  {
    file: 'Controls', title: 'Dialogs, forms and inputs', w: W,
    // A note is drawn under the tag of the name it is of. The names' row
    // scrolls, and so clips what leaves it: here it does not.
    css: '.sheet .vp .name .note.is-open{position:absolute !important;top:calc(100% + 10px) !important;left:0 !important;width:26rem;z-index:2}.sheet .vp .nametags:has(.note.is-open){overflow:visible !important;mask-image:none !important}' + LIST,
    heading: 'Dialogs, forms and inputs',
    intro: 'Everything behind a button is a popover. A dialog is a panel under a head of metal; a menu hangs from its button; a note has no head.',
    items: [
      { h2: 'Dialogs' },
      { vp: 'wide', url: vin(F150), open: ['#developers'], keep: ['#developers'], drop: ['#mcp + dl', '#mcp', '#verify + .copyable', '#verify', '#links + ul', '#links'], cap: 'Dialog', note: 'A slotted screw at each end of the head, the title struck in, Close at the far end. Sections sit over a blue rule; what is copied sits in a recessed field.' },
      { vp: 'wide', url: '/?from.fitment.number=', visit: [vin(CAMRY, '&part.id=starter'), vin(CAMRY, '&part.id=fog-lamp')], open: ['#part-types'], keep: ['#part-types'], cap: 'Part types dialog', note: 'Part-number mode\'s: every part type of the catalog under the part vocabulary\'s headings, with a filter and those opened recently. On a VIN\'s page the same list hangs from the picker.' },
      { row: [
        { vp: 'wide', box: 760, url: vin(F150), open: ['.note'], keep: ['#fit .fitstrip', '#fit .aka'], style: 'padding-bottom: 90px', cap: 'Note', note: 'How a name on a tag was reached, in words. A sure name opens it from the name itself; a doubtful one has an "i" for it.' },
        { vp: 'tab', box: 400, url: vin(F150), visit: [vin(CAMRY), vin(SILV)], open: ['#vins'], keep: ['#vins'], cap: 'Menu', note: 'Recent vehicles, each with the part type that was open on it; then the vehicles the yard holds most of.' },
      ] },
      { vp: 'wide', url: vin(F150), open: ['.card dialog'], keep: ['.card dialog'], cap: 'The check digit', note: 'The sum, a table of label-face headers.' },
      { vp: 'wide', url: vin(F150), pre: showVocab, open: ['#vocabularies'], keep: ['#vocabularies'], cap: 'Vocabularies', note: 'A dashed box around a native file input, beside what the server holds; then a table of label-face headers over a steel rule, each row with a disclosure.' },
      { vp: 'wide', base: BARE, url: '/', pre: showVocab, open: ['#vocabularies'], keep: ['#vocabularies'], cap: 'Vocabularies, none loaded', note: 'The offer: one key and a hint.' },
      { h2: 'Forms and inputs' },
      { vp: 'wide', url: vin(CAMRY), keep: ['#fit .fitstrip', '#fit .ptwrap', '#fit .aka'], cap: 'Picker', note: 'One key of the second tape on a short piece of label tape. It asks for a part type and counts the vehicle\'s.' },
      { vp: 'wide', url: vin(CAMRY), keep: ['#fit'], visit: [vin(CAMRY, '&part.id=starter'), vin(CAMRY, '&part.id=headlamp')], open: ['#part-types'], style: 'padding-bottom: 440px', cap: 'The picker\'s list', note: 'The key is pressed while its list is open. A recessed filter, then the part types opened recently, with Clear recent, then each part type as a row of the second tape with its count at the far end.' },
      { vp: 'wide', url: vin('1FTEW1EG0FFA12345'), keep: ['#answer > .hang > .cards .card.g-check', '#answer .error', '#answer .errors', '#answer .problems'], cap: 'Errors', note: 'Error-colour text on the tag, and the decoder\'s words for the problem on a strip of label tape under the answer.' },
    ],
  },
  {
    file: 'States', title: 'States', w: W,
    heading: 'States',
    intro: 'Where the page has no answer, the place stays, as a blank strip: a dashed outline with one sentence. A control in it is a key of metal; a way on is a link.',
    items: [
      { h2: 'VIN mode' },
      { vp: 'wide', url: vin('1FTEW'), keep: ['#fit'], cap: 'The model not known yet' },
      { vp: 'wide', url: vin(ACCORD), keep: ['#fit'], cap: 'No name; the catalog lacks the vehicle', note: 'A link goes to the catalog\'s lines.' },
      { vp: 'wide', url: vin(F13), keep: ['#fit'], cap: 'Nothing for its year' },
      { vp: 'wide', url: vin(F150), keep: ['#fit'], cap: 'Nothing pressed' },
      { vp: 'wide', url: vin(F150, '&part.id=wiper'), keep: ['#fit'], cap: 'A part type the vehicle has nothing for' },
      { vp: 'wide', url: vin(RAM, '&part.id=alternator'), keep: ['#fit .cands'], cap: 'A candidate with no stock', note: 'Stock is loaded and holds none of it: muted words beside a dashed rule, and nothing to press.' },
      { vp: 'wide', url: vin(F150, '&part.id=alternator&fitment.number=DEMO-1002'), keep: ['#fit .cands'], cap: 'A kind of stock with none', note: 'The shelf holds none of the open candidate: its bar stays, over "None." between asterisks.' },
      { vp: 'wide', base: BARE, url: vin(F150), keep: ['#fit'], cap: 'No vocabulary or catalog loaded', note: 'The key opens the Vocabularies dialog.' },
      { h2: 'Part-number mode' },
      { vp: 'wide', url: '/?from.fitment.number=', keep: ['#answer'], cap: 'Nothing typed yet' },
      { vp: 'wide', url: '/?from.fitment.number=DEMO-1001', keep: ['#answer'], cap: 'A number with no part type' },
      { vp: 'wide', url: '/?from.fitment.type=alternator&from.fitment.number=NOPE', keep: ['#answer'], cap: 'No such part' },
      { vp: 'wide', url: '/?from.fitment.type=mirror&from.fitment.number=DEMO-1301', keep: ['#answer'], cap: 'A part on more than one side' },
      { vp: 'wide', url: '/?from.fitment.type=alternator&from.fitment.number=DEMO-0901', keep: ['#answer'], cap: 'A retired number', note: 'The tag is the part that replaced it, stamped with the number asked for.' },
      { vp: 'wide', base: BARE, url: '/?from.fitment.number=', keep: ['#lookup', '#answer'], pre: blur, cap: 'No catalog loaded' },
      { vp: 'wide', url: vin(F150, '&from.fitment.type=alternator&from.fitment.number=DEMO-4001'), keep: ['#lookup .plate-form', '#answer'], pre: blur, cap: 'A link naming both a VIN and a part number' },
    ],
  },
];
