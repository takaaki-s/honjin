package cmd

import (
	"sort"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/takaaki-s/jind-ai/internal/action"
	"github.com/takaaki-s/jind-ai/internal/config"
	"github.com/takaaki-s/jind-ai/internal/tui"
)

var helpPopupCmd = &cobra.Command{
	Use:    "help-popup",
	Short:  "Internal: help view for tmux popup",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		configMgr, _ := config.NewManager(getConfigDir())
		var keybindings config.KeybindingsConfig
		var detachKeyHint string
		var actionPanelHint string
		var sessionFilterHint string
		var pluginHints []tui.PluginBindingHint
		if configMgr != nil {
			keybindings = configMgr.GetKeybindings()
			detachKeyHint = configMgr.GetDetachKeyHint()
			if apk := configMgr.GetActionPanelKeys(); len(apk) > 0 {
				actionPanelHint = action.FormatKeyHint(apk[0])
			}
			if sfk := configMgr.GetSessionFilterKeys(); len(sfk) > 0 {
				sessionFilterHint = action.FormatKeyHint(sfk[0])
			}
			for name, kb := range configMgr.GetPluginKeybindings() {
				if len(kb.Keys) == 0 {
					continue
				}
				pluginHints = append(pluginHints, tui.PluginBindingHint{
					KeyHint: action.FormatKeyHint(kb.Keys[0]),
					Name:    name,
				})
			}
			sort.Slice(pluginHints, func(i, j int) bool {
				return pluginHints[i].Name < pluginHints[j].Name
			})
		} else {
			keybindings = config.DefaultKeybindings()
			detachKeyHint = "Ctrl+]"
		}
		keys := tui.NewKeyMap(keybindings)

		model := tui.NewHelpModel(keys, detachKeyHint, actionPanelHint, sessionFilterHint, pluginHints)
		p := tea.NewProgram(model, tea.WithAltScreen())
		_, err := p.Run()
		return err
	},
}

func init() {
	rootCmd.AddCommand(helpPopupCmd)
}
