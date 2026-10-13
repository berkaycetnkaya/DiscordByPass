# Discord DPI Bypass (Türkiye İçin Özel & Platform Bağımsız)

Türkiye'deki internet servis sağlayıcılarının (İSS - Türk Telekom, Superonline, Turknet vb.) Discord'a uyguladığı engelleri (**DNS zehirlemesi + SNI derin paket incelemesi**) aşmak için geliştirilmiş, **VPN gerektirmeyen**, **0 ms ek gecikme (ping) oluşturan** ve **yalnızca Discord'u etkileyen** yerel vekil (proxy) sunucusu ve DPI tüneli.

---

## 🚀 Öne Çıkan Özellikler

1. **Sıfır Ping & Tam Hız (0 ms Overhead):**
   - Uzak bir VPN sunucusuna bağlanmaz.
   - Bilgisayarınız ile Discord sunucuları arasında doğrudan TCP/TLS tüneli kurar.
   - Oyunlarda veya sesli sohbetlerde asla ping artışı yapmaz.

2. **Otomatik Sistem & Güncelleme Desteği (Checking for updates Çözümü):**
   - Discord'un Rust tabanlı yerel güncelleme motoru (`reqwest`) İSS'nin DNS zehirlemesine takılmaz.
   - Program çalışırken Windows Sistem Vekili'ni otomatik aktif eder; Discord açıldığında anında güncelleme kontrolünü geçer.
   - Program kapatıldığında tüm Windows ayarlarını otomatik olarak eski haline döndürür.

3. **Platform Bağımsız (Pure Go):**
   - Windows'a özel sürücü (`WinDivert`) gerektirmez.
   - Yönetici (Admin) yetkisi istemez.
   - **Windows, macOS ve Linux** üzerinde tamamen aynı şekilde çalışır.

4. **Dahili DoH (DNS-over-HTTPS):**
   - Cloudflare (1.1.1.1) ve Google (8.8.8.8) üzerinden Discord IP'lerini çözer (İSS'nin `195.175.254.2` sahte yönlendirmesini tamamen engeller).
   - RFC standartlarına uygun TLS ClientHello kaydı parçalama motoru içerir.

---

## 📦 Nasıl Çalıştırılır?

### Masaüstünden Tek Tık:
Masaüstünüzde oluşturulan **`Discord (Bypass)`** kısayoluna çift tıklayın!

### Klasörden Başlatma:
1. [d:/Projeler/discord-bypass/start.bat](file:///d:/Projeler/discord-bypass/start.bat) dosyasına çift tıklayın.
2. Tünel açılacak ve Discord'u otomatik başlatacaktır. Canlı tünel kayıtlarını terminalde izleyebilirsiniz.
3. Çıkmak istediğinizde terminal penceresinde `CTRL + C` tuşlarına basmanız veya pencereyi kapatmanız yeterlidir.

### Manuel Kapatma / Sıfırlama:
Herhangi bir durumda servisleri tamamen durdurmak ve ayarları sıfırlamak için:
* [d:/Projeler/discord-bypass/stop.bat](file:///d:/Projeler/discord-bypass/stop.bat) dosyasına çift tıklayabilirsiniz.

---

## 📊 Canlı Web Kontrol Paneli

Uygulama açıkken tarayıcınızdan şu adrese giderek canlı durumu izleyebilirsiniz:
👉 [http://127.0.0.1:8085/](http://127.0.0.1:8085/)
