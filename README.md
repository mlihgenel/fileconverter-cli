# File Converter CLI

<p align="center">
  <img src="assets/fileconverter.gif" alt="File Converter CLI Arayüzü" width="800">
</p> 



<p align="center">
  Belgeleri, görselleri, sesleri ve videoları tamamen yerel ortamda dönüştüren modern bir CLI/TUI aracı.
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat-square&logo=go" alt="Go Version">
  <img src="https://img.shields.io/badge/License-MIT-green?style=flat-square" alt="License">
  <img src="https://img.shields.io/badge/Platform-macOS%20%7C%20Linux%20%7C%20Windows-blue?style=flat-square" alt="Platform">
  <a href="https://goreportcard.com/report/github.com/mlihgenel/fileconverter-cli"><img src="https://goreportcard.com/badge/github.com/mlihgenel/fileconverter-cli?style=flat-square" alt="Go Report Card"></a>
</p>

## İçindekiler
- [Genel Bakış](#genel-bakış)
- [Özellikler](#özellikler)
- [Kurulum](#kurulum)
- [Hızlı Başlangıç](#hızlı-başlangıç)
- [Komut Referansı](#komut-referansı)
- [Flag Referansı](#flag-referansı)
- [Desteklenen Formatlar](#desteklenen-formatlar)
- [Harici Bağımlılıklar](#harici-bağımlılıklar)
- [Yapılandırma](#yapılandırma)
- [Sorun Giderme](#sorun-giderme)
- [Geliştirme](#geliştirme)
- [Proje Yapısı](#proje-yapısı)
- [Katkı](#katkı)
- [Lisans](#lisans)

## Genel Bakış
File Converter CLI, dosya dönüştürme işlemlerini internet servislerine yükleme yapmadan yerel makinede gerçekleştiren bir komut satırı uygulamasıdır.

- Gizlilik odaklıdır: dosyalar cihazdan çıkmaz.
- İki kullanım modu sunar: CLI (otomasyon/script) ve interaktif TUI (menü tabanlı).

## Özellikler
- Belge, görsel, ses ve video dönüşümleri.
- `mp4 -> gif` dahil video dönüşümü.
- Batch dönüşüm (dizin veya glob pattern).
- Paralel işleme (`--workers`) ile yüksek performans.
- Ön izleme modu (`--dry-run`) ile risksiz batch planlama.
- Harici bağımlılık kontrolü (FFmpeg, LibreOffice, Pandoc).
- Format alias desteği (`jpeg -> jpg`, `tiff -> tif`, `markdown -> md`).

## Kurulum

### 1. Go ile kurulum (önerilen)
```bash
go install github.com/mlihgenel/fileconverter-cli@latest
```

Kurulum sonrası herhangi bir dizinden çalıştırabilmek için binary yolunun `PATH` içinde olması gerekir.
`go env GOBIN` doluysa o dizini, boşsa `$(go env GOPATH)/bin` dizinini `PATH` içine ekleyin.

### 2. PATH ayarı (herhangi bir dizinden çalıştırmak için)

#### macOS / Linux (zsh veya bash)
```bash
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.zshrc
source ~/.zshrc
```

`bash` kullanıyorsanız `~/.bashrc` veya `~/.bash_profile` dosyasına ekleyin.

#### Windows (PowerShell)
```powershell
$gopath = go env GOPATH
setx PATH "$env:PATH;$gopath\bin"
```

Ardından yeni bir terminal açın.

### 3. Kaynaktan derleme
```bash
git clone https://github.com/mlihgenel/fileconverter-cli.git
cd fileconverter-cli
go build -o fileconverter-cli .
./fileconverter-cli --help
```

Windows için:
```powershell
go build -o fileconverter-cli.exe .
.\fileconverter-cli.exe --help
```

## Hızlı Başlangıç

### Yardım menüsü
```bash
fileconverter-cli --help
fileconverter-cli help convert
fileconverter-cli help batch
fileconverter-cli help formats
```

### İnteraktif mod (TUI)
```bash
fileconverter-cli
```

### Format sorgulama
```bash
fileconverter-cli formats
fileconverter-cli formats --from mp4
fileconverter-cli formats --to gif
```

### Tek dosya dönüşümü
```bash
# Belge
fileconverter-cli convert belge.md --to pdf

# Görsel
fileconverter-cli convert fotograf.jpeg --to png

# Ses
fileconverter-cli convert ses.mp3 --to wav

# Video -> GIF
fileconverter-cli convert klip.mp4 --to gif --quality 80
```

### Toplu (batch) dönüşüm
```bash
# Dizindeki tüm .md dosyalarını PDF yap
fileconverter-cli batch ./docs --from md --to pdf

# Alt dizinlerle birlikte
fileconverter-cli batch ./videolar --from mp4 --to gif --recursive

# Ön izleme (dönüştürmeden planı gösterir)
fileconverter-cli batch ./resimler --from jpg --to webp --dry-run

# Glob kullanımı
fileconverter-cli batch "*.png" --from png --to jpg --quality 85
```

## Komut Referansı

| Komut | Ne yapar | Örnek |
|---|---|---|
| `fileconverter-cli` | İnteraktif TUI modunu başlatır | `fileconverter-cli` |
| `fileconverter-cli convert <dosya>` | Tek dosya dönüşümü | `fileconverter-cli convert input.mp4 --to gif` |
| `fileconverter-cli batch <dizin/glob>` | Toplu dönüşüm | `fileconverter-cli batch ./src --from md --to html` |
| `fileconverter-cli formats` | Desteklenen dönüşümleri listeler | `fileconverter-cli formats --from pdf` |
| `fileconverter-cli completion <shell>` | Shell completion üretir | `fileconverter-cli completion zsh` |
| `fileconverter-cli help [komut]` | Komut yardımı gösterir | `fileconverter-cli help batch` |

## Flag Referansı

### Global flag'ler

| Flag | Kısa | Açıklama |
|---|---|---|
| `--output` | `-o` | Çıktı dizini (varsayılan: kaynak dosya dizini) |
| `--verbose` | `-v` | Detaylı çıktı |
| `--workers` | `-w` | Batch modunda paralel worker sayısı |

### `convert` flag'leri

| Flag | Kısa | Açıklama |
|---|---|---|
| `--to` | `-t` | Hedef format (zorunlu) |
| `--quality` | `-q` | Kalite seviyesi (1-100) |
| `--name` | `-n` | Çıktı dosya adı (uzantısız) |

### `batch` flag'leri

| Flag | Kısa | Açıklama |
|---|---|---|
| `--from` | `-f` | Kaynak format (zorunlu) |
| `--to` | `-t` | Hedef format (zorunlu) |
| `--recursive` | `-r` | Alt dizinleri de tara |
| `--dry-run` | - | Dönüştürmeden önce planı göster |
| `--quality` | `-q` | Kalite seviyesi (1-100) |

### `formats` flag'leri

| Flag | Açıklama |
|---|---|
| `--from` | Belirli bir kaynaktan gidilebilen hedefleri listeler |
| `--to` | Belirli bir hedefe gelebilen kaynakları listeler |

## Desteklenen Formatlar

En güncel ve tam matris için:
```bash
fileconverter-cli formats
```

### Belgeler
- Kaynak/hedef: `md`, `html`, `pdf`, `docx`, `txt`, `odt`, `rtf`, `csv`
- Ek: `csv -> xlsx`

### Görseller
- `png`, `jpg/jpeg`, `webp`, `bmp`, `gif`, `tif/tiff`, `ico`

### Ses (FFmpeg)
- `mp3`, `wav`, `ogg`, `flac`, `aac`, `m4a`, `wma`, `opus`, `webm`

### Videolar (FFmpeg)
- Kaynak: `mp4`, `mov`, `mkv`, `avi`, `webm`, `m4v`, `wmv`, `flv`
- Hedef: yukarıdakiler + `gif`

## Harici Bağımlılıklar

| Araç | Ne zaman gerekir | Not |
|---|---|---|
| FFmpeg | Ses ve video dönüşümleri | `mp4 -> gif` dahil |
| LibreOffice | Bazı belge dönüşümleri (`odt/rtf/xlsx`) | Bazı dönüşümler için fallback kullanılır |
| Pandoc | Bazı Markdown belge akışları | Opsiyonel, fallback mevcut |

Uygulama interaktif modda eksik araçları kontrol eder ve kurulum için yönlendirir.

## Yapılandırma

- Konfigürasyon dosyası: `~/.fileconverter/config.json`
- Bu dosyada ilk çalıştırma bilgisi ve varsayılan çıktı dizini tutulur.
- İnteraktif moddan varsayılan çıktı dizinini değiştirebilirsiniz.

## Sorun Giderme

### `command not found: fileconverter-cli`
- `PATH` içine `$(go env GOPATH)/bin` ekleyin.
- Terminali yeniden açın.

### Eski sürüm/eskimiş help çıktısı görünüyor
```bash
cd /proje/dizini
go install .
which fileconverter-cli
fileconverter-cli --help
```

### Dönüşüm desteklenmiyor hatası
Önce formatları doğrulayın:
```bash
fileconverter-cli formats --from <kaynak>
fileconverter-cli formats --to <hedef>
```

### FFmpeg bulunamadı
macOS:
```bash
brew install ffmpeg
```
Linux (Debian/Ubuntu):
```bash
sudo apt install ffmpeg
```

## Geliştirme
```bash
git clone https://github.com/mlihgenel/fileconverter-cli.git
cd fileconverter-cli
go test ./...
go run . --help
```

## Proje Yapısı
```text
fileconverter-cli/
├── cmd/                  # Cobra komutları (convert, batch, formats, interactive)
├── internal/converter/   # Dönüştürme motorları (document, image, audio, video)
├── internal/batch/       # Worker pool ve batch yürütme
├── internal/config/      # Uygulama ayarları
├── internal/installer/   # Bağımlılık kontrol/kurulum yardımcıları
├── internal/ui/          # Ortak terminal UI yardımcıları
├── assets/               # README görselleri
└── main.go               # Uygulama giriş noktası
```

## Katkı
Katkılar memnuniyetle karşılanır.

1. Repo'yu fork edin.
2. Yeni branch açın.
3. Değişiklikleri yapın.
4. Testleri çalıştırın.
5. Pull request gönderin.

Issue ve öneriler için: [GitHub Issues](https://github.com/mlihgenel/fileconverter-cli/issues)

## Lisans
Bu proje [MIT Lisansı](LICENSE) ile lisanslanmıştır.
