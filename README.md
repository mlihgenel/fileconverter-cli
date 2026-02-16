# FileConverter CLI

<p align="center">
  <img src="assets/mainmenu.png" alt="FileConverter CLI Arayüzü" width="700">
</p>

<p align="center">
  <b>Dosyalarınızı yerel ortamda güvenli, hızlı ve kolay bir şekilde dönüştürün.</b><br>
  İnternet bağlantısı gerektirmez. Verileriniz bilgisayarınızdan asla çıkmaz.
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go" alt="Go Version">
  <img src="https://img.shields.io/badge/License-MIT-green?style=flat-square" alt="License">
  <img src="https://img.shields.io/badge/Platform-macOS%20%7C%20Linux%20%7C%20Windows-blue?style=flat-square" alt="Platform">
  <a href="https://goreportcard.com/report/github.com/mlihgenel/fileconverter-cli"><img src="https://goreportcard.com/badge/github.com/mlihgenel/fileconverter-cli?style=flat-square" alt="Go Report Card"></a>
</p>

---

## 🎯 Neden FileConverter?

Günümüzde dosya dönüştürmek için kullanılan çoğu çevrimiçi araç, dosyalarınızı sunucularına yüklemenizi gerektirir. **FileConverter**, bu işlemi tamamen kendi bilgisayarınızda yaparak gizliliğinizi ve güvenliğinizi ön planda tutar.

- **🔒 %100 Gizlilik**: Dosyalarınız hiçbir zaman internet'e yüklenmez. Tüm işlem yerel işlemcinizde gerçekleşir.
- **✨ İnteraktif Arayüz**: Karışık komutlar ezberlemenize gerek yok. Ok tuşları ile gezinebileceğiniz modern bir TUI (Terminal User Interface) sunar.
- **⚡ Yüksek Performans**: Go dilinin gücü ve paralelleştirme (worker pool) yetenekleri sayesinde binlerce dosyayı saniyeler içinde dönüştürün.
- **🛠️ Akıllı Bağımlılık Yönetimi**: Sisteminizde gerekli araçların (FFmpeg vb.) olup olmadığını kontrol eder, eksikse sizi yönlendirir.

---

## 🚀 Kurulum

### Yöntem 1: Go ile Kurulum (Önerilen)

Eğer sisteminizde Go kurulu ise, tek komutla kurabilirsiniz:

```bash
go install github.com/mlihgenel/fileconverter-cli@latest
```

### Yöntem 2: Kaynaktan Derleme

Projeyi klonlayıp kendiniz derleyebilirsiniz:

```bash
git clone https://github.com/mlihgenel/fileconverter-cli.git
cd fileconverter
go build -o fileconverter .
```

> **Not:** Kaynaktan derlediyseniz ve `GOPATH/bin` yolunda değilseniz, aşağıdaki komutları başına `./` ekleyerek çalıştırmalısınız (örneğin `./fileconverter`).

### Gereksinimler

FileConverter çoğu işlem için Go'nun standart kütüphanelerini kullanır. Ancak bazı özel formatlar için harici araçlara ihtiyaç duyar:

*   **FFmpeg**: Ses ve video dönüşümleri için gereklidir.
*   **LibreOffice / Pandoc**: (İsteğe bağlı) Bazı gelişmiş belge dönüşümleri için kullanılabilir.

Uygulama, ilk çalıştırıldığında bu araçları kontrol eder ve gerekirse kurulum için size rehberlik eder.

---

## 📖 Kullanım

### 1. İnteraktif Mod (TUI)

Hiçbir parametre vermeden çalıştırdığınızda, kullanıcı dostu interaktif arayüz açılır:

```bash
fileconverter
```

Bu modda:
*   Dosya veya klasör seçimi yapabilir,
*   Dönüştürmek istediğiniz formatı menüden seçebilir,
*   İşlem durumunu canlı progress bar ile takip edebilirsiniz.

### 2. Hızlı Komutlar (CLI)

Otomasyon veya hızlı işlemler için komut satırı argümanlarını kullanabilirsiniz.

#### Tekli Dosya Dönüşümü

```bash
# Markdown dosyasını PDF'e çevir
fileconverter convert belge.md --to pdf

# Resmi PNG formatına çevir
fileconverter convert icon.jpg --to png

# Ses dosyasını WAV formatına çevir
fileconverter convert ses.mp3 --to wav
```

#### Toplu (Batch) Dönüşüm

Klasördeki tüm dosyaları tek seferde dönüştürün:

```bash
# 'belgeler' klasöründeki tüm .md dosyalarını .html yap
fileconverter batch ./belgeler --from md --to html

# Alt klasörleri de dahil et (--recursive)
fileconverter batch ./projeler --from docx --to pdf --recursive

# Paralel işlem sayısını belirle (Hız artırma)
fileconverter batch ./fotograflar --from joy --to png --workers 8
```

---

## 📊 Desteklenen Formatlar

FileConverter çok geniş bir format yelpazesini destekler:

### 📄 Belgeler
| Kaynak | Hedef Formatlar | Notlar |
|--------|-----------------|--------|
| **MD** | HTML, PDF, DOCX, TXT | Markdown stili korunur |
| **DOCX** | PDF, TXT, MD, HTML | |
| **PDF** | TXT, HTML | Metin çıkarma odaklı |
| **HTML** | MD, TXT, PDF | |
| **TXT** | PDF, DOCX, HTML, MD | |
| **ODT** | PDF, DOCX, TXT | LibreOffice gerektirebilir |

### 🖼️ Görseller
| Kaynak | Hedef Formatlar |
|--------|-----------------|
| **PNG, JPEG, WEBP** | PNG, JPG, WEBP, GIF, BMP, TIFF, ICO |
| **BMP, TIFF, GIF** | PNG, JPG, WEBP, BMP, TIFF |

### 🎵 Ses (FFmpeg ile)
Aşağıdaki tüm formatlar arasında çapraz dönüşüm yapılabilir:
*   MP3, WAV, OGG, FLAC, AAC, M4A, WMA, OPUS

---

## ⚙️ Gelişmiş Seçenekler

| Flag | Kısa | Açıklama |
|------|-------|----------|
| `--output` | `-o` | Çıktı dosyalarının kaydedileceği dizin |
| `--verbose` | `-v` | İşlem detaylarını ekrana basar |
| `--workers` | `-w` | Batch işleminde kullanılacak thread sayısı (Varsayılan: CPU) |
| `--quality` | `-q` | Görsel kalite ayarı (1-100) |
| `--dry-run` | | İşlem yapmadan ne olacağını gösterir (Simülasyon) |

---

## 🏗️ Proje Yapısı

```
FileConverter/
├── cmd/                 # Komut satırı ve TUI mantığı (Cobra & Bubble Tea)
├── internal/
│   ├── converter/       # Dönüştürme motoru (Factory Pattern)
│   ├── batch/           # Paralel işleme (Worker Pool)
│   ├── config/          # Yapılandırma yönetimi
│   └── ui/              # Ortak UI bileşenleri
└── assets/              # Görseller ve kaynak dosyalar
```

## 🤝 Katkıda Bulunma

Katkılarınızı bekliyoruz!

1.  Bu depoyu Fork'layın.
2.  Yeni bir özellik için branch oluşturun (`git checkout -b feature/new-feature`).
3.  Değişikliklerinizi commit yapın (`git commit -m 'New feature added'`).
4.  Branch'inizi Push edin (`git push origin feature/new-feature`).
5.  Bir Pull Request oluşturun.

## 📄 Lisans

Bu proje [MIT Lisansı](LICENSE) ile lisanslanmıştır. Özgürce kullanabilir, değiştirebilir ve dağıtabilirsiniz.
