# SA/opensearch-design.md

# Amaze Search Engine - OpenSearch Design

## Mục tiêu

Thiết kế Search Document và cấu hình OpenSearch phục vụ:

* Product Search (Đa ngôn ngữ: VI, EN, TH)
* Autocomplete (Phân vùng theo Tenant)
* Spellcheck (Tối ưu hóa hiệu năng)
* Synonym Expansion (Search-time)
* Translation Expansion (Search-time & Index-time)
* Ranking (Thuật toán cân bằng)

---

# Index

`products_v1`

---

# Search Document Schema

```json
{
  "id": "uuid",
  "tenant_id": "uuid",

  "product_name_vi": "Cà phê Robusta nguyên chất",
  "product_name_en": "Pure Robusta Coffee",
  "product_name_th": "กาแฟโรบัสตาแท้",

  "product_description_vi": "Sản phẩm cà phê nguyên chất",
  "product_description_en": "Pure coffee product",
  "product_description_th": "ผลิตภัณฑ์กาแฟแท้",

  "category_name": "Coffee",
  "brand": "Amaze Coffee",
  "price": 120000.0,
  "image_url": "https://cdn.amaze.com/products/coffee.jpg",
  "inventory": 100,
  "featured": true,

  "search_tags": [
    "cà phê phin",
    "robusta việt nam",
    "cà phê đen"
  ],

  "suggest": {
    "input": [
      "cà phê",
      "coffee",
      "cafe"
    ],
    "contexts": {
      "tenant_context": ["d04cbce0-d67d-42b6-aa26-178eb2011d98"]
    }
  }
}
```

---

# OpenSearch Mappings & Analyzers

Để tối ưu hóa tìm kiếm tự nhiên, các trường văn bản được phân tách bằng các bộ phân tích (Analyzers) bản địa:

* **`product_name_vi` & `product_description_vi`**: Sử dụng analyzer tiếng Việt (ví dụ: `vi_analyzer` thông qua plugin `analysis-vietnamese` hoặc bộ phân tích chuẩn hóa dấu tiếng Việt).
* **`product_name_en` & `product_description_en`**: Sử dụng analyzer `english` có sẵn của OpenSearch (hỗ trợ Porter Stemmer để chuyển các từ dạng số nhiều, thì quá khứ về từ gốc).
* **`product_name_th` & `product_description_th`**: Sử dụng analyzer `thai` có sẵn của OpenSearch (hỗ trợ phân tách câu tiếng Thái không có khoảng trắng thông qua từ điển tích hợp sẵn).
* **`search_tags`**: Định nghĩa là trường `text` sử dụng analyzer tương ứng với ngôn ngữ gốc của sản phẩm để hỗ trợ phân tách từ khóa mở rộng do AI sinh ra.
* **`image_url`**: Định nghĩa là `keyword` với `"index": false` để tránh tốn tài nguyên đánh chỉ mục không cần thiết, chỉ dùng để hiển thị.

---

# Search Fields & Query Boosting

Khi người dùng tìm kiếm, Search API sẽ truy vấn trên các trường ngôn ngữ tương ứng với ngôn ngữ của UI hoặc ngôn ngữ được detect từ câu truy vấn.

## Trọng số tìm kiếm (Field Weights)

* **Ưu tiên cao nhất (High Weight)**:
  `product_name_{lang}`
  boost: 5
* **Ưu tiên trung bình (Medium Weight)**:
  `category_name`
  boost: 3
  
  `search_tags` (Các từ khóa mở rộng do AI sinh ra)
  boost: 2.5
* **Ưu tiên thấp (Low Weight)**:
  `product_description_{lang}`
  boost: 1

---

# Ranking Rules

Điểm số cuối cùng của sản phẩm được tính dựa trên tổng hợp điểm tương quan văn bản (TF-IDF/BM25) và điểm thúc đẩy nghiệp vụ (Business Boosting):

1. **Exact Match**: So khớp chính xác từ khóa được boost điểm cao hơn so với khớp một phần.
2. **Phrase Match**: So khớp cả cụm từ chính xác đứng cạnh nhau được ưu tiên.
3. **Synonym Match**: So khớp từ đồng nghĩa (được dịch từ Search API sang câu query OR).
4. **Translation Match**: So khớp trên các trường ngôn ngữ phụ (ví dụ: tìm tiếng Anh trên sản phẩm gốc tiếng Việt).
5. **Featured Product**: Sử dụng `function_score` để boost điểm số nhân thêm một trọng số cố định nếu `"featured": true`.
6. **Inventory**: Áp dụng hàm giảm thiểu (decay/sigmoid function) đối với sản phẩm hết hàng (`inventory = 0`) để đẩy chúng xuống cuối kết quả tìm kiếm một cách mượt mà, thay vì ẩn hoàn toàn.

---

# Autocomplete (Hỗ trợ Multi-tenancy)

Sử dụng **Completion Suggester** trên trường `suggest`.

Để đảm bảo tính năng gợi ý từ khóa độc lập giữa các Marketplace (Tenant), cấu hình **Context Suggester** trong mappings:

```json
"suggest": {
  "type": "completion",
  "contexts": [
    {
      "name": "tenant_context",
      "type": "category",
      "path": "tenant_id"
    }
  ]
}
```

* **Truy vấn**: Khi gọi API Autocomplete, Search API bắt buộc truyền kèm tham số ngữ cảnh `tenant_context` mang giá trị `tenant_id` của phiên hiện tại để lọc từ khóa gợi ý chính xác.

---

# Spellcheck (Tối ưu hóa độ trễ)

Hệ thống sử dụng cơ chế kiểm tra lỗi chính tả 2 tầng:

1. **Tầng 1 (Local Cache)**: Tra cứu nhanh từ khóa trong từ điển chính tả đã được Admin phê duyệt (`spellcheck_dictionary`) đã lưu tại Redis/PostgreSQL. Nếu khớp, tự động sửa đổi từ khóa trước khi truy vấn OpenSearch.
2. **Tầng 2 (OpenSearch Term Suggester)**: Nếu không có trong cache, sử dụng `Term Suggestion` của OpenSearch.
   * *Tối ưu hóa*: Để tránh 2 lần gọi mạng riêng biệt, Search API sử dụng API **Multi-Search (`_msearch`)** của OpenSearch để thực hiện đồng thời truy vấn tìm kiếm sản phẩm và truy vấn gợi ý sửa lỗi trong **duy nhất một request**.

---

# Synonym Expansion (Search-time)

Thay vì lưu từ đồng nghĩa cố định vào Document lúc index, hệ thống áp dụng cơ chế mở rộng ở thời điểm tìm kiếm (Search-time Expansion):

1. Người dùng tìm kiếm từ khóa `coffee`.
2. Search API tra cứu từ điển `synonyms` (đã được cache ở Redis).
3. Hệ thống phát hiện từ đồng nghĩa: `cafe`, `cà phê`.
4. Câu query gửi tới OpenSearch được viết lại dưới dạng:
   `(product_name_en:coffee) OR (product_name_en:cafe) OR (product_name_vi:cà phê)` với các mức boost tương ứng.
5. **Lợi ích**: Admin có thể cập nhật từ điển đồng nghĩa liên tục trong Admin UI và có hiệu lực lập tức mà không cần re-index lại bất kỳ tài liệu nào.

---

# Translation Expansion (Search-time & Ingestion-time)

* **Ingestion-time**: Khi tạo/cập nhật sản phẩm, hệ thống dịch sẵn tên và mô tả sản phẩm sang tiếng Anh và tiếng Thái rồi lưu vào các trường tương ứng (`product_name_en`, `product_name_th`).
* **Search-time**: Đối với từ khóa tìm kiếm của khách hàng, Search API sẽ tra cứu nhanh từ điển dịch thuật tĩnh (`translations`). Nếu từ khóa nằm trong từ điển, câu truy vấn sẽ được mở rộng sang trường ngôn ngữ tương ứng để tăng tỷ lệ tìm thấy sản phẩm.
