# Swift Search Engine - OpenSearch Design

## Mục tiêu

Thiết kế tài liệu lưu trữ (Search Document), cấu hình lập chỉ mục (Mappings & Analyzers) và thiết lập truy vấn trong OpenSearch nhằm phục vụ:

* **Product Search**: Tìm kiếm đa ngôn ngữ (Tiếng Việt, Tiếng Anh, Tiếng Thái)
* **Autocomplete**: Gợi ý tự động hoàn thành từ khóa tức thời dựa trên n-gram và phân vùng theo `tenant_id`
* **Spellcheck**: Gợi ý sửa lỗi chính tả từ truy vấn thông qua Phrase Suggester
* **Synonym & Translation Expansion**: Mở rộng từ đồng nghĩa và từ dịch tại Search-time
* **Ranking Engine**: Xếp hạng kết quả tìm kiếm kết hợp độ tương quan văn bản và các yếu tố kinh doanh (featured, inventory)

---

# Chỉ mục (Index)

Hệ thống sử dụng cấu trúc Alias:
* **Tên Index vật lý**: `products_v1`
* **Tên Alias sử dụng trong ứng dụng**: `products`

---

# Cấu trúc tài liệu (Search Document Schema)

Tài liệu được lưu trữ dạng phẳng (flat) chứa tất cả bản dịch và thông tin cần thiết trong cùng một document:

```json
{
  "id": "7a3b4e9f-9c02-4d22-8f1a-b3d5c6e7f8a9",
  "tenant_id": "d04cbce0-d67d-42b6-aa26-178eb2011d98",
  "category_id": "c12a3b4c-5d6e-7f8a-9b0c-1d2e3f4a5b6c",
  "category_name": "Cà phê túi lọc",
  
  "product_name_vi": "Cà phê Robusta nguyên chất",
  "product_name_en": "Pure Robusta Coffee",
  "product_name_th": "กาแฟโรบัสตาแท้",
  
  "description_vi": "Sản phẩm cà phê nguyên chất, hương vị đậm đà từ vùng Tây Nguyên.",
  "description_en": "Pure coffee product with rich flavor from the Central Highlands.",
  "description_th": "ผลิตภัณฑ์กาแฟแท้รสชาติเข้มข้นจากที่ราบสูงตอนกลาง",
  
  "brand": "Swift Coffee",
  "price": 120000.0,
  "image_url": "https://cdn.swift.com/products/coffee.jpg",
  "inventory": 100,
  "featured": true,
  "status": "active",
  
  "suggest": "Cà phê Robusta nguyên chất Pure Robusta Coffee กาแฟโรบัสตาแท้"
}
```
*Trường `suggest` được ghép từ tất cả các tên sản phẩm đã dịch, cách nhau bằng khoảng trắng.*

---

# Cấu hình OpenSearch Settings & Mappings

### 1. Phân tích ngữ nghĩa (Analyzers & Filters)
* **`vi_ascii_analyzer`**: Bộ phân tích tùy chỉnh sử dụng tokenizer `standard` kết hợp filter `lowercase` và `asciifolding`. Giúp chuyển toàn bộ chữ có dấu tiếng Việt về không dấu và chữ thường, hỗ trợ tìm kiếm không dấu khớp có dấu.
* **`autocomplete_analyzer`**: Bộ phân tích n-gram tùy chỉnh (min_gram: 2, max_gram: 10). Dùng để phân tách chuỗi `suggest` thành các cụm ký tự ngắn phục vụ tính năng autocomplete khi người dùng đang gõ phím.

### 2. Định nghĩa Mapping Chi tiết

```json
{
  "settings": {
    "analysis": {
      "filter": {
        "autocomplete_filter": {
          "type": "ngram",
          "min_gram": 2,
          "max_gram": 10
        }
      },
      "analyzer": {
        "vi_ascii_analyzer": {
          "type": "custom",
          "tokenizer": "standard",
          "filter": [
            "lowercase",
            "asciifolding"
          ]
        },
        "autocomplete_analyzer": {
          "type": "custom",
          "tokenizer": "standard",
          "filter": [
            "lowercase",
            "asciifolding",
            "autocomplete_filter"
          ]
        }
      }
    },
    "index": {
      "number_of_shards": 1,
      "number_of_replicas": 0,
      "max_ngram_diff": 8
    }
  },
  "mappings": {
    "properties": {
      "id": { "type": "keyword" },
      "tenant_id": { "type": "keyword" },
      "category_id": { "type": "keyword" },
      "category_name": { "type": "text", "analyzer": "vi_ascii_analyzer" },
      "product_name_vi": { "type": "text", "analyzer": "vi_ascii_analyzer" },
      "product_name_en": { "type": "text", "analyzer": "english" },
      "product_name_th": { "type": "text", "analyzer": "thai" },
      "description_vi": { "type": "text", "analyzer": "vi_ascii_analyzer" },
      "description_en": { "type": "text", "analyzer": "english" },
      "description_th": { "type": "text", "analyzer": "thai" },
      "brand": { "type": "keyword" },
      "price": { "type": "double" },
      "image_url": { "type": "keyword" },
      "inventory": { "type": "integer" },
      "featured": { "type": "boolean" },
      "status": { "type": "keyword" },
      "suggest": {
        "type": "text",
        "analyzer": "autocomplete_analyzer",
        "search_analyzer": "vi_ascii_analyzer"
      }
    }
  }
}
```

---

# Quy tắc tìm kiếm & Phân bổ Trọng số (Query Boosting)

Truy vấn tìm kiếm sử dụng cấu trúc **Multi-Match Query** kết hợp hệ số tăng cường (Boosting) tùy chỉnh dựa trên ngôn ngữ tìm kiếm hiện tại (`X-Language-Key`).

### Trọng số các trường theo từng ngôn ngữ (Target Fields)

#### 1. Ngôn ngữ tìm kiếm là Tiếng Việt (`vi`)
* `product_name_vi` boost `5.0` (Ưu tiên cao nhất)
* `product_name_en` boost `1.5`
* `product_name_th` boost `1.5`
* `category_name` boost `3.0`
* `description_vi` boost `1.0`
* `description_en` boost `0.5`
* `description_th` boost `0.5`
* `brand` (mặc định)
* `suggest` (mặc định)

#### 2. Ngôn ngữ tìm kiếm là Tiếng Anh (`en`)
* `product_name_en` boost `5.0`
* `product_name_vi` boost `1.5`
* `product_name_th` boost `1.5`
* `category_name` boost `3.0`
* `description_en` boost `1.0`
* `description_vi` boost `0.5`
* `description_th` boost `0.5`
* `brand` (mặc định)
* `suggest` (mặc định)

#### 3. Ngôn ngữ tìm kiếm là Tiếng Thái (`th`)
* `product_name_th` boost `5.0`
* `product_name_en` boost `1.5`
* `product_name_vi` boost `1.5`
* `category_name` boost `3.0`
* `description_th` boost `1.0`
* `description_en` boost `0.5`
* `description_vi` boost `0.5`
* `brand` (mặc định)
* `suggest` (mặc định)

---

# Giải thuật Xếp hạng (Ranking Rules)

Hệ thống sử dụng cấu trúc truy vấn `function_score` kết hợp điểm số BM25 và các tiêu chí xếp hạng nghiệp vụ:

1. **Exact Phrase Boost**: Thực hiện so khớp cụm từ đầy đủ (`match_phrase`) trên tên sản phẩm của ngôn ngữ hiện tại và cộng điểm boost:
   * Boost `5.0` cho `product_name_vi`
   * Boost `3.0` cho `product_name_en`
   * Boost `3.0` cho `product_name_th`
2. **Synonym Boost**: Các từ khóa được mở rộng (Synonym) được cấu hình hệ số boost thấp hơn (`0.6`) để tránh làm loãng kết quả tìm kiếm chính xác.
3. **Featured Boost**: Nếu sản phẩm được đánh dấu `"featured": true`, điểm số cuối cùng được nhân thêm một hệ số thúc đẩy (ví dụ: `1.2`).
4. **Inventory Decay**: Nếu sản phẩm hết hàng (`inventory = 0`), điểm số của sản phẩm sẽ bị nhân với một hệ số suy giảm (ví dụ: `0.5`) nhằm đẩy sản phẩm xuống cuối trang kết quả tìm kiếm thay vì ẩn hẳn.

---

# Autocomplete (Hỗ trợ Multi-tenancy)

Để tối ưu hóa hiệu năng, tính năng gợi ý từ khóa không dùng Completion Suggester của OpenSearch mà sử dụng truy vấn tiêu chuẩn trên trường `suggest` kết hợp lọc ngữ cảnh:

```json
{
  "size": 10,
  "query": {
    "bool": {
      "must": [
        {
          "term": {
            "tenant_id": "<tenant_id>"
          }
        },
        {
          "bool": {
            "should": [
              {
                "match": {
                  "suggest": {
                    "query": "<từ-khóa>",
                    "operator": "and",
                    "boost": 2.0
                  }
                }
              },
              {
                "multi_match": {
                  "query": "<từ-khóa>",
                  "fields": [
                    "product_name_vi^5.0",
                    "product_name_en^1.5",
                    "product_name_th^1.5",
                    "brand"
                  ],
                  "operator": "and"
                }
              }
            ],
            "minimum_should_match": 1
          }
        }
      ]
    }
  }
}
```
*Nhờ filter `tenant_id`, dữ liệu gợi ý của các tenant/marketplace khác nhau sẽ hoàn toàn cô lập.*

---

# Spellcheck (Sửa lỗi chính tả bằng Phrase Suggester)

Khi thực hiện tìm kiếm sản phẩm, Search API chèn trực tiếp yêu cầu gợi ý chính tả (`suggest` block) vào truy vấn OpenSearch để nhận lại kết quả sửa lỗi trong cùng một request:

```json
{
  "suggest": {
    "suggest_vi": {
      "text": "<từ-khóa>",
      "phrase": {
        "field": "product_name_vi",
        "size": 1,
        "confidence": 0.8,
        "direct_generator": [
          {
            "field": "product_name_vi",
            "suggest_mode": "missing"
          }
        ]
      }
    },
    "suggest_en": {
      "text": "<từ-khóa>",
      "phrase": {
        "field": "product_name_en",
        "size": 1,
        "confidence": 0.8,
        "direct_generator": [
          {
            "field": "product_name_en",
            "suggest_mode": "missing"
          }
        ]
      }
    }
  }
}
```
*Nếu từ gốc bị lỗi chính tả (ví dụ: `iphne`), OpenSearch sẽ trả về từ sửa lỗi gợi ý `iphone` để hiển thị cho người dùng.*
