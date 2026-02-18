package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mlihgenel/fileconverter-cli/internal/converter"
)

var resizePresetsCmd = &cobra.Command{
	Use:   "resize-presets",
	Short: "Hazır boyutlandırma presetlerini listeler",
	Long: `Görsel ve video boyutlandırma için kullanılabilecek hazır ölçüleri gösterir.

Örnek:
  fileconverter-cli resize-presets
  fileconverter-cli convert video.mp4 --to mp4 --preset story --resize-mode pad`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Hazır Boyut Presetleri")
		fmt.Println("----------------------")
		for _, p := range converter.ResizePresets() {
			fmt.Printf("  %-20s %4dx%-4d  %s\n", p.Name, p.Width, p.Height, p.Description)
		}

		fmt.Println()
		fmt.Println("İpuçları:")
		fmt.Println("  - Manuel ölçü için: --width <değer> --height <değer> --unit px|cm")
		fmt.Println("  - Hazır preset yerine direkt ölçü de verebilirsiniz: --preset 1080x1920")
		fmt.Println("  - Yatay videoyu dikeye güvenli çevirmek için: --preset story --resize-mode pad")
	},
}

func init() {
	rootCmd.AddCommand(resizePresetsCmd)
}
