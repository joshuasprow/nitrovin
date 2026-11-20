// The six frames of the design canvas (docs/design/canvas): what each one
// shows, as specimens of the page. A specimen names a width class (vp), a
// URL, the elements to keep (their ancestors stay, their siblings go), and
// any popover to draw open. Every URL is the demo set's.
import { BARE } from './engine.mjs';
const F150 = '1FTEW1EG7FFA12345', ACCORD = '1HGCM82633A004352', F13 = '1FTFW1EF6DFA12345', CAMRY = '4T1B11HK7JU512345', SILV = '1GCVKREC6EZ123456';
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
const DIAGRAM = ['#lookup .plate-form', '#answer > .under', '#answer > .leaders', '#answer > .hang'];

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
      { vp: 'wide', url: '/?from.fitment.number=', keep: ['#lookup'], pre: blur, cap: 'Part-number mode, empty', note: 'The second mode: a part type key and a number field, each under its struck label.' },
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
    intro: 'Three materials, the same in both colour schemes: brushed aluminum, vinyl tape and manila. Three heights: static tape is stuck on; a control stands raised on a steel ledge; pressed, it sits flush, 2px down and darker.',
    items: [
      { h2: 'Aluminum' },
      { vp: 'wide', url: vin(F150), keep: ['header', '#lookup'], pre: blur, mark: { '.open-developers': 'is-active' }, cap: 'Plates and keys', note: 'The brand is a static plate. A button is a key of the same metal, raised; Developers is shown pressed, as is the VIN key of the switch.' },
      { h2: 'Vinyl tape' },
      { vp: 'wide', url: vin(F150, '&part.id=alternator'), keep: ['#answer > .hang > .aka', '#fit .carry', '#fit .fitstrip', '#fit .ptwrap', '#fit .stems', '#fit .runs', '#fit .drops'], cap: 'Label tape, the second tape, pinstripe', note: 'Black label tape carries names, letters raised white. The second tape is the plate\'s metal with the tape\'s ridges: the part type picker is a key of it on its own piece of label tape, and a part type opened recently is a strip of it, raised. Lines are pinstripe tape in a group\'s colour, lit on one edge, and run straight into the tape.' },
      { h2: 'Manila' },
      { vp: 'wide', url: vin(F150, '&part.id=alternator&fitment.number=DEMO-1001'), keep: ['#answer > .leaders', '#answer > .hang > .cards', '#fit .cands', '#fit .rails'], cap: 'Tags', note: 'Paper stock with tooth, clipped corners, a reinforced eyelet, a band printed in the group\'s colour. A control on a tag is label tape; a candidate\'s stock strip is the one metal on a tag.' },
    ],
  },
  {
    file: 'Diagram', title: 'Cards, brackets and lines', w: W,
    heading: 'Cards, brackets and lines',
    intro: 'Each card is a manila tag hung by its eyelet from its group\'s line. A row is a label over its value, both left-aligned. Under the cards, the names are one block of label tape; the Vehicle and Model year lines run through it and on down the page.',
    items: [
      { vp: 'wide', url: vin(F150), keep: DIAGRAM, pre: blur, cap: 'Decoded', note: 'Brackets, leader lines, cards, join lines and the names.' },
      { vp: 'wide', url: vin('1FTEW'), keep: DIAGRAM, pre: blur, cap: 'Waiting', note: 'A group not yet typed keeps its place: a faint bracket and line, and a dashed outline where its tag will hang.' },
      { row: [
        { vp: 'tab', url: vin(F150), keep: DIAGRAM, pre: blur, cap: 'Narrow', note: 'Below 64rem the cards and the names are one sideways-scrolling row; no leader lines.' },
        { vp: 'phone', url: vin(F150), keep: DIAGRAM, pre: blur, cap: 'Phone', note: 'The same row at 390.' },
      ] },
    ],
  },
  {
    file: 'Fitment', title: 'The fitment card', w: W,
    heading: 'The fitment card',
    intro: 'The parts that fit: a strip naming the catalog\'s vehicle, a picker of its part types, and the candidates as manila tags on rails that come down from the picker. A candidate\'s stock strip opens its shelf and yard rows.',
    items: [
      { vp: 'wide', url: vin(F150, '&part.id=alternator&fitment.number=DEMO-1001'), keep: ['#answer > .hang > .aka', '#fit'], cap: 'Picker, candidates, marks and stock', note: 'Alternator pressed: the picker names it and counts its candidates, and × beside it clears. Under its tape, what the decode says of this vehicle, to match each candidate\'s options against. The first candidate\'s stock is open under its row.' },
      { vp: 'wide', url: vin(F150, '&part.id=alternator'), open: ['#part-types'], keep: ['#answer > .hang > .aka', '#fit'], style: 'padding-bottom: 300px', cap: 'The picker\'s list', note: 'Hung from the picker at its width, over the candidates: every part type under the part vocabulary\'s headings, in an order that never changes. The pressed one is flush; pressing it again changes nothing.' },
      { vp: 'wide', url: vin(SILV, '&part.id=brake-rotor-rear'), keep: ['#answer > .hang > .aka', '#fit'], cap: 'By redirect', note: 'One tag: the rails run from the riser to its eyelet. The mark names the part type the catalog lists it under.' },
      { row: [
        { vp: 'tab', url: vin(F150, '&part.id=alternator'), keep: ['#answer > .hang > .aka', '#fit'], cap: 'Narrow', note: 'Three tags across down to 34rem. The names scroll; the strip does not.' },
        { vp: 'phone', url: vin(F150, '&part.id=alternator&fitment.number=DEMO-1001'), keep: ['#answer > .hang > .aka', '#fit'], cap: 'Phone', note: 'One sideways row of tags; open rows sit under their tag, shelf over yard.' },
      ] },
      { vp: 'wide', url: '/?from.fitment.type=alternator&from.fitment.number=DEMO-1001&fitment.number=DEMO-1001', keep: ['#answer'], cap: 'Part-number mode', note: 'The part\'s tag, its stock under it, and the vehicles it fits as the catalog names them.' },
    ],
  },
  {
    file: 'Controls', title: 'Dialogs, forms and inputs', w: W,
    css: '.sheet .vp .tagplate > .note.is-open{position:absolute !important;top:calc(100% + 10px) !important;right:0 !important;left:auto !important;width:30rem;z-index:2}',
    heading: 'Dialogs, forms and inputs',
    intro: 'Everything behind a button is a popover. A dialog is a panel under a head of metal; a menu hangs from its button; a note has no head.',
    items: [
      { h2: 'Dialogs' },
      { vp: 'wide', url: vin(F150), open: ['#developers'], keep: ['#developers'], drop: ['#mcp + dl', '#mcp', '#verify + .copyable', '#verify', '#links + ul', '#links'], cap: 'Dialog', note: 'A slotted screw at each end of the head, the title struck in, Close at the far end. Sections sit over a blue rule; what is copied sits in a recessed field.' },
      { vp: 'wide', url: '/?from.fitment.number=', visit: [vin(CAMRY, '&part.id=starter'), vin(CAMRY, '&part.id=fog-lamp')], open: ['#part-types'], keep: ['#part-types'], cap: 'Part types dialog', note: 'Part-number mode\'s: every part type of the catalog under the part vocabulary\'s headings, with a filter and those opened recently. On a VIN\'s page the same list hangs from the picker.' },
      { row: [
        { vp: 'wide', box: 760, style: 'padding-bottom: 80px', url: vin(F150), open: ['.note'], keep: ['#answer > .hang > .aka', '.note'], cap: 'Note', note: 'How a name on the tape was reached.' },
        { vp: 'tab', box: 400, url: vin(F150), visit: [vin(CAMRY), vin(SILV)], open: ['#vins'], keep: ['#vins'], cap: 'Menu', note: 'Recent VINs, then the examples.' },
      ] },
      { vp: 'wide', url: vin(F150), open: ['.card dialog'], keep: ['.card dialog'], cap: 'The check digit', note: 'The sum, a table of label-face headers.' },
      { vp: 'wide', url: vin(F150), pre: showVocab, open: ['#vocabularies'], keep: ['#vocabularies'], cap: 'Vocabularies', note: 'A dashed box around a native file input, beside what the server holds; then a table of label-face headers over a steel rule, each row with a disclosure.' },
      { vp: 'wide', base: BARE, url: '/', pre: showVocab, open: ['#vocabularies'], keep: ['#vocabularies'], cap: 'Vocabularies, none loaded', note: 'The offer: one key and a hint.' },
      { h2: 'Forms and inputs' },
      { vp: 'wide', url: vin(CAMRY), keep: ['#fit .fitstrip', '#fit .ptwrap'], visit: [vin(CAMRY, '&part.id=starter'), vin(CAMRY, '&part.id=headlamp')], cap: 'Picker and Recent', note: 'The picker asks for a part type and counts the vehicle\'s. Up to three opened recently sit beside it on its tape.' },
      { vp: 'wide', url: vin(CAMRY), keep: ['#fit .fitstrip', '#fit .ptwrap', '#part-types'], open: ['#part-types'], style: 'padding-bottom: 560px', cap: 'The picker\'s list', note: 'A recessed filter, then each part type as a row of the second tape with its count at the far end.' },
      { vp: 'wide', url: vin('1FTEW1EG0FFA12345'), keep: ['#answer > .hang > .cards .card.g-check', '#answer .error', '#answer .errors', '#answer .problems'], cap: 'Errors', note: 'Error-colour text on the tag and under the names.' },
    ],
  },
  {
    file: 'States', title: 'States', w: W,
    heading: 'States',
    intro: 'Where the page has no answer, the place stays, as a blank strip: a dashed outline with one sentence. A control in it is a key of metal; a way on is a link.',
    items: [
      { h2: 'VIN mode' },
      { vp: 'wide', url: vin('1FTEW'), keep: ['#answer > .hang > .aka', '#fit'], cap: 'The model not known yet' },
      { vp: 'wide', url: vin(ACCORD), keep: ['#answer > .hang > .aka', '#fit'], cap: 'No name; the catalog lacks the vehicle', note: 'A link goes to the catalog\'s lines.' },
      { vp: 'wide', url: vin(F13), keep: ['#answer > .hang > .aka', '#fit'], cap: 'Nothing for its year' },
      { vp: 'wide', url: vin(F150), keep: ['#answer > .hang > .aka', '#fit'], cap: 'Nothing pressed' },
      { vp: 'wide', url: vin(F150, '&part.id=wiper'), keep: ['#fit'], cap: 'A part type the vehicle has nothing for' },
      { vp: 'wide', base: BARE, url: vin(F150), keep: ['#answer > .hang > .aka', '#fit'], cap: 'No vocabulary or catalog loaded', note: 'The key opens the Vocabularies dialog.' },
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
