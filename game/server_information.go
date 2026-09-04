package game

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
)

var (
	serverInformationColor = color.RGBA{R: 255, G: 255, B: 99, A: 255}
	skillMessageColor      = color.RGBA{R: 255, G: 155, B: 155, A: 255}
)

func (m *WorldMode) applyBossInfo(info network.BossInfo) {
	name := strings.TrimSpace(info.Name)
	switch info.Kind {
	case network.BossInfoNotOnMap:
		m.ui.minimap.ClearBossMarker()
		m.ui.console.AddErrorMessage("Boss monster not found.")
	case network.BossInfoAlive:
		m.ui.minimap.SetBossMarker(int(info.X), int(info.Y))
	case network.BossInfoAliveAnnounced:
		m.ui.minimap.SetBossMarker(int(info.X), int(info.Y))
		m.ui.console.AddColoredMessage(serverInformationColor, "%s is on this map.", bossDisplayName(name))
	case network.BossInfoDead:
		m.ui.minimap.ClearBossMarker()
		m.ui.console.AddColoredMessage(serverInformationColor, "%s", bossRespawnMessage(name, info.MinRespawn, info.MaxRespawn))
	}
}

func (m *WorldMode) applySkillMessage(message network.SkillMessage, now time.Time) {
	text, ok := skillMessageText(message.ID)
	if !ok {
		glog.Debugf("unknown skill message id=%d", message.ID)
		return
	}
	m.ui.console.AddColoredMessage(skillMessageColor, "%s", text)
	m.ui.poptips.Show(text, now)
}

func skillMessageText(id int32) (string, bool) {
	switch id {
	case 0x15:
		return "All abnormal status effects have been removed.", true
	case 0x16:
		return "You are immune to all abnormal status effects.", true
	case 0x17:
		return "Max HP +100%.", true
	case 0x18:
		return "Max SP +100%.", true
	case 0x19:
		return "All stats +20.", true
	case 0x1c:
		return "Your weapon is enchanted with the Holy element.", true
	case 0x1d:
		return "Your armor is enchanted with the Holy element.", true
	case 0x1e:
		return "DEF +25%.", true
	case 0x1f:
		return "ATK +100%.", true
	case 0x20:
		return "HIT and Flee +50.", true
	case 0x28:
		return "The coating protected the equipment from Full Strip.", true
	default:
		return "", false
	}
}

func bossRespawnMessage(name string, minimum, maximum time.Duration) string {
	name = bossDisplayName(name)
	if maximum > minimum && maximum > 0 {
		return fmt.Sprintf("%s will respawn between %s and %s.", name, formatBossRespawnDuration(minimum), formatBossRespawnDuration(maximum))
	}
	if minimum <= 0 {
		return fmt.Sprintf("%s will respawn shortly.", name)
	}
	return fmt.Sprintf("%s will respawn in %s.", name, formatBossRespawnDuration(minimum))
}

func bossDisplayName(name string) string {
	if name == "" {
		return "The boss"
	}
	return name
}

func formatBossRespawnDuration(duration time.Duration) string {
	if duration <= 0 {
		return "less than a minute"
	}
	hours := int(duration / time.Hour)
	minutes := int(duration%time.Hour) / int(time.Minute)
	switch {
	case hours == 0:
		return fmt.Sprintf("%d minute(s)", minutes)
	case minutes == 0:
		return fmt.Sprintf("%d hour(s)", hours)
	default:
		return fmt.Sprintf("%d hour(s) and %d minute(s)", hours, minutes)
	}
}
