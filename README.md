# SpoofDPI Turkiye

Read in other Languages: [🇹🇷Turkish](https://github.com/esenmx/SpoofDPI-Turkiye), [🇬🇧English](https://github.com/esenmx/SpoofDPI-Turkiye/blob/main/_docs/README_en.md)

Spoof DPI'ın bu sürümü **Türkiye'de** kullanılmak üzere yapılandırılmıştır.

![image](https://user-images.githubusercontent.com/45588457/148035986-8b0076cc-fefb-48a1-9939-a8d9ab1d6322.png)

## Kurulum

Direkt olarak [releases](https://github.com/esenmx/SpoofDPI-Turkiye/releases) bölümünden indirebilir veya
[Buradan](https://github.com/esenmx/SpoofDPI-Turkiye/blob/main/_docs/INSTALL.md) kurulum aşamalarını takip edebilirsiniz.

## Kullanım

Programımız Türkiye'ye özel olarak konfigure edildiği için sizin için uygun sürümü direkt olarak başlatarak çalıştırabilirsiniz.

### Gelişmiş Kullanım

```text
Kullanım: spoofdpi [seçenekler...]
  -addr string
        dinleme adresi (varsayılan "127.0.0.1")
  -port value
        port (varsayılan 8080)
  -dns-addr string
        birincil dns sunucusu (varsayılan "1.1.1.1")
  -dns-port value
        dns için port numarası (varsayılan 53)
  -dns-fallback value
        ek dns sunucuları; -dns-addr ile paralel sorgulanır; birden çok kez verilebilir
        (varsayılan: 8.8.8.8, 9.9.9.9)
  -dns-ipv4-only
        sadece IPv4 adreslerini çözer
  -enable-doh
        DNS-over-HTTPS'i ek bir çözümleyici olarak etkinleştirir (varsayılan true)
  -doh-url string
        DoH uç noktası URL'si (varsayılan "https://cloudflare-dns.com/dns-query")
  -doh-bootstrap-ip string
        DoH için doğrudan aranacak IP; sistem çözümleyicisini atlar (varsayılan "1.1.1.1")
  -pattern value
        DPI'yı yalnızca bu regex deseniyle eşleşen paketlerde atlar; birden çok kez verilebilir
  -window-size value
        Parçalanmış istemci dönüşü için bayt sayısı cinsinden yığın boyutu,
        varsayılan değer DPI'ı atlamazsa daha düşük değerler deneyin;
        verilmediğinde, istemci dönüş paketi iki parça halinde gönderilecektir:
        ilk veri paketi için parçalama ve geri kalanı şeklinde (varsayılan 5)
  -timeout value
        milisaniye cinsinden zaman aşımı; verilmediğinde zaman aşımı olmaz
  -system-proxy
        sistem genelinde proxy aktif et (yalnızca macOS; diğer platformlarda göz ardı edilir)
  -silent
        başlangıçta afişi ve sunucu bilgilerini gösterme
  -debug
        hata ayıklamayı aktif eder
  -v    spoofdpi'nin sürümünü yazdırır; bu, diğer bazı ilgili bilgileri içerebilir
```

ISP'ye özel ince ayar için: [`_docs/TR_DPI.md`](_docs/TR_DPI.md).

> Chrome tarayıcısında Hotspot Shield gibi herhangi bir vpn uzantısı kullanıyorsanız,
  Ayarlar > Eklentiler, bölümüne gidin ve onları devre dışı bırakın.

### OSX

`Spoofdpi`ı çalıştırdığınızda proxy'nizi otomatik olarak ayarlayacaktır

### Linux

`Spoofdpi`ı çalıştırın ve favori tarayıcınızı proxy seçeneği ile açın

```bash
google-chrome --proxy-server="http://127.0.0.1:8080"
```

## Nasıl Çalışır

### HTTP

 Dünyadaki çoğu web sitesi artık HTTPS'yi desteklediğinden, SpoofDPI HTTP istekleri için Derin Paket Denetimlerini atlamaz, ancak yine de tüm HTTP istekleri için proxy bağlantısı sunar.

### HTTPS

 TLS her handshake işlemini şifrelese de, İstemci dönüş paketinde alan adları hala düz metin olarak gösterilir.
 Başka bir deyişle, başka biri pakete baktığında, paketin nereye gittiğini kolayca tahmin edebilir.
 DPI işlenirken alan adı önemli bilgiler sunabilir ve aslında İstemci dönüş paketini gönderdikten hemen sonra bağlantının engellendiğini görebiliriz.
 Bunu aşmak için bazı yollar denedim ve İstemci dönüş paketini parçalara bölerek gönderdiğimizde yalnızca ilk parçanın denetlendiğini fark ettim.
 SpoofDPI'ın bunu atlamak için yaptığı şey, bir isteğin ilk 1 baytını sunucuya göndermektir,
 ve sonra geri kalanını gönder.

## Benzer Projeler

[GoodbyeDPI-Turkey](https://github.com/cagritaskn/GoodbyeDPI-Turkey) @cagritaskn (Windows)
