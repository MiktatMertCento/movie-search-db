# Movie Search DB - Kurulum Rehberi

Bu proje; Go backend, React (Vite) frontend ve pgvector destekli PostgreSQL veritabanı kullanarak film arama ve embedding işlemlerini gerçekleştiren bir sistemdir.

## 1. Gereksinimler

* **Docker & Docker Desktop:** Konteyner yönetimi için gereklidir.
* **Ollama:** Yerel AI modellerini çalıştırmak için gereklidir.
* **TMDB API Key:** Film verilerini güncellemek için gereklidir.

### Ollama Yapılandırması ve Model Kurulumu
1. **Modeli İndir:** Terminal üzerinden embedding modelini çek:
   ```bash
   ollama pull bge-m3
   ```
2. **Dış Erişim Ayarı (Windows):** Docker konteynerlerinin ana makinedeki Ollama'ya erişebilmesi için:
    - Sistem ortam değişkenlerine `OLLAMA_HOST` ekle.
    - Değerini `0.0.0.0` yap.
    - Ollama uygulamasını sağ alttan (System Tray) kapatıp yeniden başlat.

## 2. Ortam Değişkenleri (.env)

Kök dizinde yer alan `.env` dosyasını şu şekilde yapılandır:

```env
DB_HOST=db
DB_PORT=5432
DB_USER=root
DB_PASSWORD=randompassword*2026.0
DB_NAME=movie_ai
DB_SSLMODE=disable

OLLAMA_BASE_URL=http://host.docker.internal:11434
TMDB_API_KEY=e5aff83cba4c85311d5a39c070e5178a
```

## 3. Kurulum ve Çalıştırma

Terminalde proje kök dizinine git ve sistemi inşa et:

```bash
docker compose up -d --build
```

## 4. Servis Akışı (Başlatma Hiyerarşisi)

Sistem, bağımlılıkları yönetmek için şu sırayla ayağa kalkar:

1. **db:** `pgvector` destekli PostgreSQL veritabanı başlatılır.
2. **setup:** Veritabanı hazır olduğunda (`healthy`) şu scriptleri sırasıyla çalıştırır:
    - `seeder.go`: `datas/` altındaki CSV dosyalarını veritabanına aktarır.
    - `updater.go`: TMDB API üzerinden güncel verileri çeker.
    - `embedder.go`: `bge-m3` modelini kullanarak vektörleri oluşturur.
    - İşlem bittiğinde `setup_done.lock` dosyası oluşturur ve servis durur.
3. **backend:** Setup servisi başarıyla kapandığında Go sunucusu başlar.
4. **frontend:** Backend hazır olduğunda React uygulaması sunulur.

## 5. Erişim Portları

| Servis | Adres |
| :--- | :--- |
| **Frontend (UI)** | `http://localhost:3000` |
| **Backend (API)** | `http://localhost:8080` |
| **PostgreSQL** | `localhost:5432` |

## 6. Kritik Komutlar

```bash
# Tüm servislerin loglarını izle
docker compose logs -f

# Sadece veri işleme sürecini (setup) takip et
docker compose logs -f setup

# Veritabanını ve tüm konteynerleri sıfırla (Volume dahil)
docker compose down -v && docker compose up -d --build
```

---
**Geliştiren:** Miktat Mert.