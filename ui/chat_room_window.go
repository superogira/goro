package ui

import (
	"fmt"
	"strings"

	"github.com/gogpu/ui/core/scrollview"
	"github.com/gogpu/ui/core/textfield"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/state"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	chatRoomWindowW        = 320
	chatRoomWindowContentH = 152
	chatRoomMessagePadX    = 8
	chatRoomMessagePadY    = 6
	chatRoomMessageGap     = 1
)

type ChatRoomWindowAction struct {
	Message string
	Leave   bool
}

type chatRoomLine struct {
	text  string
	color widget.Color
	kind  string // "" normal, "sys" system, "err" error — for the DOM twin
}

type ChatRoomWindow struct {
	Window
	title      string
	limit      uint16
	public     bool
	count      uint16
	owner      string
	members    []string
	input      string
	inputField *textfield.Widget
	lines      []chatRoomLine
	scrollY    state.Signal[float32]
	action     ChatRoomWindowAction
	// webOpen marks the DOM-panel twin as the active presentation (web
	// build only); webSig skips no-op syncs so the page keeps focus.
	webOpen bool
	webSig  string
}

func (w *ChatRoomWindow) Open(ctx Context, title string, limit uint16, public bool, members []string) {
	title = strings.TrimSpace(title)
	if title == "" {
		return
	}
	w.ctx = ctx
	w.title = title
	w.limit = limit
	w.public = public
	w.members = compactChatRoomMembers(members)
	w.count = uint16(len(w.members))
	w.owner = firstChatRoomMember(w.members)
	w.input = ""
	w.inputField = nil
	w.lines = nil
	w.scrollY = nil
	w.action = ChatRoomWindowAction{}
	if chatRoomWebSync(w.webState(true)) {
		w.webOpen = true
		w.webSig = w.webSignature()
		w.open = true
		return
	}
	w.EnsureWindow(chatRoomWindowW, ROWindowTitleHeight+chatRoomWindowContentH+ROWindowFooterHeight)
	w.SetBackground(widget.RGBA8(0, 0, 0, 0))
	w.SetFullRedraw(true)
	w.Window.Open(ctx, w.widgetTree(ctx))
	w.focusInput()
	w.Publish(ctx)
}

// Close hides whichever presentation is active.
func (w *ChatRoomWindow) Close() {
	if w.webOpen {
		w.webOpen = false
		w.open = false
		w.webSig = ""
		chatRoomWebSync(w.webState(false))
		return
	}
	w.Window.Close()
}

func (w *ChatRoomWindow) Update(ctx Context) bool {
	w.ctx = ctx
	if !w.IsOpen() {
		return false
	}
	if w.webOpen {
		return w.updateWeb(ctx)
	}
	w.EnsureWindow(chatRoomWindowW, ROWindowTitleHeight+chatRoomWindowContentH+ROWindowFooterHeight)
	w.SetFullRedraw(true)
	if ctx.Input != nil && ctx.Input.JustPressed(input.KeyEscape) {
		// Leaving a chat room is server state, not just a local window close.
		// Match the title-bar close button and queue the exit request.
		w.requestLeave(ctx)
		return true
	}
	if w.submitFromFocusedEnter(ctx) {
		w.Publish(ctx)
		return true
	}
	consumed := w.Window.Update(ctx)
	w.Publish(ctx)
	return consumed
}

// updateWeb services the DOM panel: sends and leaves arrive through the
// action queue. It stays unconsumed while idle so the game keeps normal
// input — the page owns the typing focus, like the whisper twin.
func (w *ChatRoomWindow) updateWeb(ctx Context) bool {
	message, leave := drainChatRoomWebActions()
	if leave {
		w.requestLeave(ctx)
		return true
	}
	if message != "" {
		w.action = ChatRoomWindowAction{Message: message}
	}
	if ctx.Input != nil && ctx.Input.JustPressed(input.KeyEscape) {
		w.requestLeave(ctx)
		return true
	}
	return false
}

func (w *ChatRoomWindow) Rebind(ctx Context) {
	if w.webOpen {
		return
	}
	w.EnsureWindow(chatRoomWindowW, ROWindowTitleHeight+chatRoomWindowContentH+ROWindowFooterHeight)
	w.SetBackground(widget.RGBA8(0, 0, 0, 0))
	w.SetFullRedraw(true)
	if !w.IsOpen() {
		return
	}
	w.ctx = ctx
	w.inputField = nil
	w.refresh(ctx)
}

func (w *ChatRoomWindow) PopAction() ChatRoomWindowAction {
	action := w.action
	w.action = ChatRoomWindowAction{}
	return action
}

// RoomTitle/RoomLimit/RoomPublic/RoomCount expose the live room state so
// the game layer can mirror the owner's board locally when the server
// transfers ownership to us.
func (w *ChatRoomWindow) RoomTitle() string { return w.title }
func (w *ChatRoomWindow) RoomLimit() uint16 { return w.limit }
func (w *ChatRoomWindow) RoomPublic() bool  { return w.public }
func (w *ChatRoomWindow) RoomCount() uint16 {
	if w.count > 0 {
		return w.count
	}
	return uint16(len(w.members))
}

func (w *ChatRoomWindow) AddMessage(ctx Context, text string) {
	w.addLine(ctx, strings.TrimSpace(text), "", widget.RGBA8(232, 238, 245, 255))
}

func (w *ChatRoomWindow) AddSystem(ctx Context, text string) {
	w.addLine(ctx, strings.TrimSpace(text), "sys", widget.RGBA8(160, 190, 230, 255))
}

func (w *ChatRoomWindow) AddError(ctx Context, text string) {
	w.addLine(ctx, strings.TrimSpace(text), "err", Color(ErrorTextColor))
}

func (w *ChatRoomWindow) SetMembers(ctx Context, members []string, owner string) {
	w.members = compactChatRoomMembers(members)
	w.count = uint16(len(w.members))
	w.owner = strings.TrimSpace(owner)
	if w.owner == "" {
		w.owner = firstChatRoomMember(w.members)
	}
	if w.IsOpen() {
		w.refresh(ctx)
	}
}

func (w *ChatRoomWindow) AddMember(ctx Context, name string, count uint16) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	if !chatRoomHasMember(w.members, name) {
		w.members = append(w.members, name)
	}
	w.count = count
	if w.count == 0 {
		w.count = uint16(len(w.members))
	}
	if w.owner == "" {
		w.owner = firstChatRoomMember(w.members)
	}
	if w.IsOpen() {
		w.refresh(ctx)
	}
}

func (w *ChatRoomWindow) RemoveMember(ctx Context, name string, count uint16) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	next := w.members[:0]
	for _, member := range w.members {
		if !strings.EqualFold(member, name) {
			next = append(next, member)
		}
	}
	w.members = next
	w.count = count
	if w.count == 0 {
		w.count = uint16(len(w.members))
	}
	if strings.EqualFold(w.owner, name) {
		w.owner = firstChatRoomMember(w.members)
	}
	if w.IsOpen() {
		w.refresh(ctx)
	}
}

func (w *ChatRoomWindow) SetOwner(ctx Context, name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	w.owner = name
	if !chatRoomHasMember(w.members, name) {
		w.members = append([]string{name}, w.members...)
		w.count = uint16(len(w.members))
	}
	if w.IsOpen() {
		w.refresh(ctx)
	}
}

func (w *ChatRoomWindow) UpdateRoom(ctx Context, title string, limit uint16, count uint16, public bool) {
	title = strings.TrimSpace(title)
	if title != "" {
		w.title = title
	}
	w.limit = limit
	w.count = count
	w.public = public
	if w.IsOpen() {
		w.refresh(ctx)
	}
}

func (w *ChatRoomWindow) addLine(ctx Context, text string, kind string, color widget.Color) {
	if text == "" {
		return
	}
	w.lines = append(w.lines, chatRoomLine{text: text, color: color, kind: kind})
	if len(w.lines) > 80 {
		copy(w.lines, w.lines[len(w.lines)-80:])
		w.lines = w.lines[:80]
	}
	if w.IsOpen() {
		w.refresh(ctx)
	}
}

func (w *ChatRoomWindow) widgetTree(ctx Context) widget.Widget {
	return Win(
		Title(w.windowTitle()),
		CloseButton(true),
		OnClose(func() { w.requestLeave(ctx) }),
		Size(chatRoomWindowW, ROWindowTitleHeight+chatRoomWindowContentH+ROWindowFooterHeight),
		Background(widget.RGBA8(0, 0, 0, 0)),
		Content(w.contentTree()),
		Footer(w.footerTree(ctx)...),
	)
}

func (w *ChatRoomWindow) contentTree() widget.Widget {
	messageHeight := chatRoomWindowContentH - 24 - chatRoomMessagePadY*2
	textWidth := maxInt(1, scrollbarSafeIntWidth(chatRoomWindowW-2*chatRoomMessagePadX)-6)
	lines := wrapChatRoomLines(w.visibleLines(), textWidth)
	contentHeight := whisperMessageContentHeight(len(lines))
	rows := make([]widget.Widget, 0, len(lines)+1)
	if spacerH := messageHeight - contentHeight - chatRoomMessageGap; spacerH > 0 {
		rows = append(rows, primitives.Box().Height(float32(spacerH)))
	}
	for _, line := range lines {
		rows = append(rows,
			primitives.Box(
				rotheme.Text(line.text).
					Color(line.color).
					MaxLines(1),
			).Height(consoleLineH),
		)
	}
	messageList := primitives.Box(rows...).
		Gap(chatRoomMessageGap).
		CrossAlign(primitives.CrossAxisStretch)
	w.ensureScrollSignal().Set(consoleBottomScrollY(len(lines), messageHeight))
	return primitives.Box(
		primitives.Box(w.memberSummary()).
			Height(20).
			PaddingXY(chatRoomMessagePadX, 3).
			CrossAlign(primitives.CrossAxisStretch),
		primitives.Box(
			scrollview.New(
				primitives.Box(messageList).
					PaddingRight(ROScrollbarGutter),
				scrollview.ScrollbarOpt(scrollview.ScrollbarAuto),
				scrollview.ScrollYSignal(w.ensureScrollSignal()),
				scrollview.ScrollStep(float32(consoleLineH*3)),
			),
		).
			Height(float32(messageHeight+chatRoomMessagePadY*2)).
			PaddingXY(chatRoomMessagePadX, chatRoomMessagePadY).
			CrossAlign(primitives.CrossAxisStretch),
	).
		Background(widget.RGBA8(14, 18, 24, 188)).
		BorderStyle(1, widget.RGBA8(180, 198, 218, 95)).
		CrossAlign(primitives.CrossAxisStretch)
}

func (w *ChatRoomWindow) memberSummary() widget.Widget {
	count := int(w.count)
	if count <= 0 {
		count = len(w.members)
	}
	limit := int(w.limit)
	if limit <= 0 {
		limit = count
	}
	names := chatRoomMemberListText(w.members, w.owner)
	if names == "" {
		names = "No members"
	}
	text := fmt.Sprintf("%d/%d  %s", count, limit, names)
	return rotheme.Text(trimRunes(text, 58)).
		Color(widget.RGBA8(181, 222, 239, 255)).
		MaxLines(1).
		Ellipsis()
}

func (w *ChatRoomWindow) footerTree(ctx Context) []widget.Widget {
	return []widget.Widget{
		primitives.Expanded(
			primitives.Box(w.inputWidget(ctx)).
				Height(24).
				CrossAlign(primitives.CrossAxisStretch),
		),
		rotheme.Button("Send", func() {
			w.submit(ctx)
		}),
		rotheme.Button("Leave", func() {
			w.requestLeave(ctx)
		}),
	}
}

func (w *ChatRoomWindow) inputWidget(ctx Context) *textfield.Widget {
	if w.inputField != nil {
		return w.inputField
	}
	w.inputField = rotheme.TextField(
		w.input,
		textfield.TypeText,
		func(value string) {
			w.input = value
		},
		func(string) {
			w.submit(ctx)
		},
		textfield.MaxLength(100),
		textfield.Placeholder("Message"),
	)
	w.focusInput()
	return w.inputField
}

func (w *ChatRoomWindow) submit(ctx Context) {
	message := strings.TrimSpace(w.input)
	if message == "" {
		return
	}
	w.action = ChatRoomWindowAction{Message: message}
	w.input = ""
	if w.inputField != nil {
		w.inputField.SetText("")
	}
	w.focusInput()
	w.refresh(ctx)
}

func (w *ChatRoomWindow) requestLeave(ctx Context) {
	w.action = ChatRoomWindowAction{Leave: true}
	w.Close()
	w.Publish(ctx)
}

func (w *ChatRoomWindow) submitFromFocusedEnter(ctx Context) bool {
	if ctx.Input == nil || w.inputField == nil || !w.inputField.IsFocused() {
		return false
	}
	if !ctx.Input.JustPressed(input.KeyEnter) {
		return false
	}
	w.submit(ctx)
	return true
}

func (w *ChatRoomWindow) refresh(ctx Context) {
	if w.webOpen {
		// The DOM twin owns the presentation; republishing here would
		// flash a canvas window on top of it.
		w.syncWebIfChanged()
		return
	}
	w.SetContent(w.widgetTree(ctx))
	w.focusInput()
	w.Publish(ctx)
	// No app-wide Invalidate here: SetContent already invalidates the
	// window's damage rect, and UIApp.Invalidate() maps to the full canvas.
}

// webState snapshots the room for the DOM twin.
func (w *ChatRoomWindow) webState(open bool) chatRoomWebState {
	lines := make([]whisperWebLine, 0, len(w.lines))
	for _, line := range w.lines {
		lines = append(lines, whisperWebLine{text: line.text, kind: line.kind})
	}
	return chatRoomWebState{
		open:    open,
		title:   w.title,
		public:  w.public,
		limit:   w.limit,
		count:   w.count,
		owner:   w.owner,
		members: w.members,
		lines:   lines,
	}
}

// webSignature changes whenever anything the panel renders changed, so
// idle frames skip the sync and the page keeps its input focus.
func (w *ChatRoomWindow) webSignature() string {
	last := ""
	if len(w.lines) > 0 {
		last = w.lines[len(w.lines)-1].text
	}
	return fmt.Sprintf("%s|%t|%d|%d|%s|%s|%d|%s",
		w.title, w.public, w.limit, w.count, w.owner,
		strings.Join(w.members, ","), len(w.lines), last)
}

func (w *ChatRoomWindow) syncWebIfChanged() {
	if sig := w.webSignature(); sig != w.webSig {
		w.webSig = sig
		chatRoomWebSync(w.webState(true))
	}
}

func (w *ChatRoomWindow) focusInput() {
	if w.inputField != nil {
		w.inputField.SetFocused(true)
	}
}

func (w *ChatRoomWindow) ensureScrollSignal() state.Signal[float32] {
	if w.scrollY == nil {
		w.scrollY = state.NewSignal[float32](0)
	}
	return w.scrollY
}

func (w *ChatRoomWindow) windowTitle() string {
	privacy := "Public"
	if !w.public {
		privacy = "Private"
	}
	return fmt.Sprintf("%s - %s", trimRunes(w.title, 24), privacy)
}

func (w *ChatRoomWindow) visibleLines() []chatRoomLine {
	if len(w.lines) == 0 {
		return []chatRoomLine{{text: "No messages", color: widget.RGBA8(150, 165, 182, 255)}}
	}
	return w.lines
}

func compactChatRoomMembers(members []string) []string {
	out := make([]string, 0, len(members))
	for _, member := range members {
		member = strings.TrimSpace(member)
		if member != "" {
			out = append(out, member)
		}
	}
	return out
}

func firstChatRoomMember(members []string) string {
	if len(members) == 0 {
		return ""
	}
	return strings.TrimSpace(members[0])
}

func chatRoomHasMember(members []string, name string) bool {
	for _, member := range members {
		if strings.EqualFold(member, name) {
			return true
		}
	}
	return false
}

func chatRoomMemberListText(members []string, owner string) string {
	owner = strings.TrimSpace(owner)
	labels := make([]string, 0, len(members))
	for _, member := range members {
		member = strings.TrimSpace(member)
		if member == "" {
			continue
		}
		if owner != "" && strings.EqualFold(member, owner) {
			member += " (owner)"
		}
		labels = append(labels, member)
	}
	return strings.Join(labels, ", ")
}
