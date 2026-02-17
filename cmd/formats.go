package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mlihgenel/fileconverter-cli/internal/converter"
	"github.com/mlihgenel/fileconverter-cli/internal/ui"
)

var (
	formatsFrom string
	formatsTo   string
)

var formatsCmd = &cobra.Command{
	Use:   "formats",
	Short: "Desteklenen formatları ve dönüşümleri listele",
	Long: `Tüm desteklenen dosya formatlarını ve aralarındaki dönüşüm yollarını gösterir.

Örnekler:
  fileconverter-cli formats
  fileconverter-cli formats --from pdf
  fileconverter-cli formats --to docx`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if formatsFrom != "" {
			return showConversionsFrom(formatsFrom)
		}
		if formatsTo != "" {
			return showConversionsTo(formatsTo)
		}
		return showAllFormats()
	},
}

func showAllFormats() error {
	pairs := converter.GetAllConversions()
	if len(pairs) == 0 {
		ui.PrintWarning("Hiç dönüşüm kaydedilmemiş.")
		return nil
	}

	// Kategorilere ayır
	docPairs := filterByCategory(pairs, "document")
	audioPairs := filterByCategory(pairs, "audio")
	imgPairs := filterByCategory(pairs, "image")

	fmt.Println()
	fmt.Printf("  %s %s%sDesteklenen Dönüşümler%s\n", "📋", ui.Bold, ui.Cyan, ui.Reset)
	fmt.Println()

	if len(docPairs) > 0 {
		fmt.Printf("  %s %sBelge Formatları%s\n", ui.IconFile, ui.Bold, ui.Reset)
		printPairsTable(docPairs)
		fmt.Println()
	}

	if len(audioPairs) > 0 {
		fmt.Printf("  %s %sSes Formatları%s (FFmpeg gerektirir)\n", ui.IconAudio, ui.Bold, ui.Reset)
		if !converter.IsFFmpegAvailable() {
			ui.PrintWarning("FFmpeg kurulu değil! Ses dönüşümleri çalışmaz.")
			fmt.Printf("    Kurulum: %sbrew install ffmpeg%s (macOS)\n", ui.Yellow, ui.Reset)
		}
		printPairsTable(audioPairs)
		fmt.Println()
	}

	if len(imgPairs) > 0 {
		fmt.Printf("  %s %sGörsel Formatları%s\n", ui.IconImage, ui.Bold, ui.Reset)
		printPairsTable(imgPairs)
		fmt.Println()
	}

	// Özet
	totalPairs := len(pairs)
	formats := converter.GetAllFormats()
	fmt.Printf("  %sToplam: %d format, %d dönüşüm yolu%s\n\n",
		ui.Dim, len(formats), totalPairs, ui.Reset)

	return nil
}

func showConversionsFrom(from string) error {
	from = converter.NormalizeFormat(from)
	pairs := converter.GetConversionsFrom(from)

	if len(pairs) == 0 {
		ui.PrintWarning(fmt.Sprintf("'%s' formatından yapılabilecek dönüşüm bulunamadı.", from))
		return nil
	}

	fmt.Println()
	icon := ui.PrintFormatCategory(from)
	fmt.Printf("  %s %s%s formatından dönüştürülebilir:%s\n\n", icon, ui.Bold, strings.ToUpper(from), ui.Reset)

	headers := []string{"Hedef Format", "Açıklama"}
	var rows [][]string
	for _, p := range pairs {
		rows = append(rows, []string{strings.ToUpper(p.To), p.Description})
	}
	ui.PrintTable(headers, rows)
	fmt.Println()

	return nil
}

func showConversionsTo(to string) error {
	to = converter.NormalizeFormat(to)
	pairs := converter.GetConversionsTo(to)

	if len(pairs) == 0 {
		ui.PrintWarning(fmt.Sprintf("'%s' formatına dönüştürülebilecek kaynak bulunamadı.", to))
		return nil
	}

	fmt.Println()
	icon := ui.PrintFormatCategory(to)
	fmt.Printf("  %s %s%s formatına dönüştürülebilir:%s\n\n", icon, ui.Bold, strings.ToUpper(to), ui.Reset)

	headers := []string{"Kaynak Format", "Açıklama"}
	var rows [][]string
	for _, p := range pairs {
		rows = append(rows, []string{strings.ToUpper(p.From), p.Description})
	}
	ui.PrintTable(headers, rows)
	fmt.Println()

	return nil
}

func printPairsTable(pairs []ConversionPairSort) {
	headers := []string{"Kaynak", "Hedef", "Açıklama"}
	var rows [][]string
	for _, p := range pairs {
		rows = append(rows, []string{
			strings.ToUpper(p.From),
			strings.ToUpper(p.To),
			p.Description,
		})
	}
	ui.PrintTable(headers, rows)
}

// ConversionPairSort sıralanabilir pair
type ConversionPairSort = converter.ConversionPair

func filterByCategory(pairs []converter.ConversionPair, category string) []ConversionPairSort {
	docFormats := map[string]bool{"md": true, "html": true, "pdf": true, "docx": true, "txt": true}
	audioFormats := map[string]bool{"mp3": true, "wav": true, "ogg": true, "flac": true, "aac": true, "m4a": true, "wma": true}
	imgFormats := map[string]bool{"png": true, "jpg": true, "webp": true, "bmp": true, "gif": true, "tif": true}

	var filtered []ConversionPairSort
	for _, p := range pairs {
		switch category {
		case "document":
			if docFormats[p.From] && docFormats[p.To] {
				filtered = append(filtered, p)
			}
		case "audio":
			if audioFormats[p.From] {
				filtered = append(filtered, p)
			}
		case "image":
			if imgFormats[p.From] {
				filtered = append(filtered, p)
			}
		}
	}

	// Sırala
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].From != filtered[j].From {
			return filtered[i].From < filtered[j].From
		}
		return filtered[i].To < filtered[j].To
	})

	return filtered
}

func init() {
	formatsCmd.Flags().StringVar(&formatsFrom, "from", "", "Bu formattan hangi formatlara dönüşüm yapılabilir")
	formatsCmd.Flags().StringVar(&formatsTo, "to", "", "Bu formata hangi formatlardan dönüşüm yapılabilir")

	rootCmd.AddCommand(formatsCmd)
}
