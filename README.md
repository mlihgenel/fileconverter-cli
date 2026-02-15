# FileConverter CLI

<p align="center">
  <b>Dosyalarınızı yerel ortamda güvenli bir şekilde dönüştürün.</b><br>
  Belge, ses ve görsel dosyalarını internet'e yüklemeden, tamamen yerel olarak farklı formatlara dönüştürün.
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat-square&logo=go" alt="Go Version">
  <img src="https://img.shields.io/badge/License-MIT-green?style=flat-square" alt="License">
  <img src="https://img.shields.io/badge/Platform-macOS%20%7C%20Linux%20%7C%20Windows-blue?style=flat-square" alt="Platform">
</p>

---

## ✨ Özellikler

- 🔒 **%100 Yerel** — Dosyalarınız hiçbir zaman internet'e yüklenmez
- ⚡ **Hızlı** — Go ile yazılmış, optimize edilmiş performans
- 📦 **Toplu Dönüşüm** — Worker pool ile paralel batch dönüşüm
- 📄 **Belge Formatları** — MD, HTML, PDF, DOCX, TXT
- 🎵 **Ses Formatları** — MP3, WAV, OGG, FLAC, AAC, M4A, WMA
- 🖼️ **Görsel Formatları** — PNG, JPEG, WEBP, BMP, GIF, TIFF
- 🎨 **Kullanıcı Dostu** — Renkli çıktı, progress bar, emoji ikonlar
- 🐚 **Shell Completion** — Bash, Zsh, Fish, PowerShell desteği

## 📋 Gereksinimler

- **Go 1.21+** (derlemek için)
- **FFmpeg** (yalnızca ses dönüşümleri için gerekli)

### FFmpeg Kurulumu (opsiyonel)

```bash
# macOS
brew install ffmpeg

# Ubuntu/Debian
sudo apt install ffmpeg

# Windows (Chocolatey)
choco install ffmpeg
```

## 🚀 Kurulum

### Go ile Kurulum

```bash
go install github.com/melihgenel/fileconverter@latest
```

### Kaynaktan Derleme

```bash
git clone https://github.com/melihgenel/fileconverter.git
cd fileconverter
go build -o fileconverter .
```

## 📖 Kullanım

### Tekli Dönüşüm

```bash
# Markdown → PDF
fileconverter convert README.md --to pdf

# Markdown → HTML
fileconverter convert belge.md --to html

# Markdown → DOCX
fileconverter convert rapor.md --to docx

# PDF → Plain Text
fileconverter convert dosya.pdf --to txt

# Görsel dönüşüm (kalite ayarı ile)
fileconverter convert resim.png --to jpg --quality 90

# Ses dönüşüm
fileconverter convert muzik.mp3 --to wav

# Çıktı dizini belirtme
fileconverter convert dosya.md --to pdf --output ./cikti/

# Çıktı dosya adı belirtme
fileconverter convert dosya.md --to pdf --name sonuc
```

### Toplu Dönüşüm (Batch)

```bash
# Dizindeki tüm MD dosyalarını PDF'e dönüştür
fileconverter batch ./belgeler --from md --to pdf

# Alt dizinleri de dahil et
fileconverter batch ./belgeler --from md --to pdf --recursive

# Çıktıyı farklı dizine yaz
fileconverter batch ./belgeler --from md --to html --output ./cikti/

# Worker sayısını ayarla
fileconverter batch ./resimler --from png --to jpg --workers 8

# Kalite ayarı ile
fileconverter batch ./resimler --from png --to jpg --quality 85

# Ön izleme (dry-run)
fileconverter batch ./belgeler --from md --to pdf --dry-run
```

### Desteklenen Formatları Görüntüleme

```bash
# Tüm desteklenen formatlar
fileconverter formats

# Belirli bir formattan yapılabilecek dönüşümler
fileconverter formats --from pdf

# Belirli bir formata yapılabilecek dönüşümler
fileconverter formats --to docx
```

## 📊 Desteklenen Dönüşümler

### 📄 Belge Formatları

| Kaynak | Hedef Formatlar |
|--------|-----------------|
| MD | HTML, PDF, TXT, DOCX |
| HTML | TXT, MD |
| PDF | TXT |
| DOCX | TXT |
| TXT | PDF, HTML, DOCX |

### 🎵 Ses Formatları (FFmpeg gerektirir)

MP3, WAV, OGG, FLAC, AAC, M4A, WMA — tüm formatlar arası çapraz dönüşüm (42 yol)

### 🖼️ Görsel Formatları

| Kaynak | Hedef Formatlar |
|--------|-----------------|
| PNG | JPG, BMP, GIF, TIFF |
| JPEG | PNG, BMP, GIF, TIFF |
| WEBP | PNG, JPG, BMP, GIF, TIFF |
| BMP | PNG, JPG, GIF, TIFF |
| GIF | PNG, JPG, BMP, TIFF |
| TIFF | PNG, JPG, BMP, GIF |

**Toplam: 18 format, 78 dönüşüm yolu**

## ⚙️ Global Seçenekler

| Flag | Kısa | Açıklama |
|------|-------|----------|
| `--verbose` | `-v` | Detaylı çıktı modu |
| `--output` | `-o` | Çıktı dizini |
| `--workers` | `-w` | Paralel worker sayısı (varsayılan: CPU çekirdek sayısı) |
| `--version` | | Versiyon bilgisi |
| `--help` | `-h` | Yardım |

## 🏗️ Proje Yapısı

```
FileConverter/
├── main.go                          # Giriş noktası
├── cmd/
│   ├── root.go                      # Root komut, global flag'ler
│   ├── convert.go                   # Tekli dönüşüm
│   ├── batch.go                     # Toplu dönüşüm
│   └── formats.go                   # Format listesi
├── internal/
│   ├── converter/
│   │   ├── converter.go             # Interface + Registry
│   │   ├── document.go              # Belge dönüşümleri
│   │   ├── audio.go                 # Ses dönüşümleri (FFmpeg)
│   │   └── image.go                 # Görsel dönüşümleri
│   ├── batch/
│   │   └── pool.go                  # Worker Pool
│   └── ui/
│       └── progress.go              # Progress bar, renkli çıktı
├── go.mod
└── go.sum
```

## 🤝 Katkıda Bulunma

1. Fork yapın
2. Feature branch oluşturun (`git checkout -b feature/yeni-format`)
3. Değişikliklerinizi commit edin (`git commit -m 'Yeni format desteği eklendi'`)
4. Branch'e push edin (`git push origin feature/yeni-format`)
5. Pull Request açın

### Yeni Converter Ekleme

`internal/converter/` dizininde yeni bir dosya oluşturun ve `Converter` interface'ini implemente edin:

```go
package converter

type MyConverter struct{}

func init() {
    Register(&MyConverter{})
}

func (c *MyConverter) Name() string { return "My Converter" }
func (c *MyConverter) SupportsConversion(from, to string) bool { /* ... */ }
func (c *MyConverter) SupportedConversions() []ConversionPair { /* ... */ }
func (c *MyConverter) Convert(input, output string, opts Options) error { /* ... */ }
```

## 📄 Lisans

MIT License — detaylar için [LICENSE](LICENSE) dosyasına bakın.
