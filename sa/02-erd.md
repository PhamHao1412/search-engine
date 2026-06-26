# SA/erd.md

# Amaze Search Engine - ERD

## Mục tiêu

Thiết kế dữ liệu phục vụ Product Management, Search Analytics và AI Suggestion Engine.

---

# 1. products

| Column            | Type      |
| ----------------- | --------- |
| id                | uuid      |
| tenant_id         | uuid      |
| name              | varchar   |
| description       | text      |
| category_id       | uuid      |
| brand             | varchar   |
| price             | decimal   |
| image_url         | varchar   |
| inventory         | int       |
| status            | varchar   |
| featured          | boolean   |
| original_language | varchar   |
| created_at        | timestamp |
| updated_at        | timestamp |

---

# 2. categories

| Column     | Type      |
| ---------- | --------- |
| id         | uuid      |
| tenant_id  | uuid      |
| name       | varchar   |
| parent_id  | uuid      |
| created_at | timestamp |
| updated_at | timestamp |

---

# 3. search_logs

| Column           | Type      |
| ---------------- | --------- |
| id               | uuid      |
| tenant_id        | uuid      |
| query            | varchar   |
| normalized_query | varchar   |
| result_count     | int       |
| searched_at      | timestamp |

---

# 4. click_logs

| Column        | Type      |
| ------------- | --------- |
| id            | uuid      |
| tenant_id     | uuid      |
| search_log_id | uuid      |
| query         | varchar   |
| product_id    | uuid      |
| clicked_at    | timestamp |

---

# 5. synonyms

| Column     | Type      |
| ---------- | --------- |
| id         | uuid      |
| tenant_id  | uuid      |
| keyword    | varchar   |
| synonym    | varchar   |
| status     | varchar   |
| created_at | timestamp |

Ví dụ:

keyword:

coffee

synonym:

cafe

---

# 6. translations

| Column          | Type    |
| --------------- | ------- |
| id              | uuid    |
| tenant_id       | uuid    |
| source_language | varchar |
| target_language | varchar |
| source_text     | varchar |
| translated_text | varchar |

Ví dụ:

coffee

cà phê

---

# 7. spellcheck_dictionary

| Column       | Type      |
| ------------ | --------- |
| id           | uuid      |
| tenant_id    | uuid      |
| typo_word    | varchar   |
| correct_word | varchar   |
| status       | varchar   |
| created_at   | timestamp |

Ví dụ:

typo_word: iphne
correct_word: iphone

---

# 8. ai_suggestions

| Column           | Type      |
| ---------------- | --------- |
| id               | uuid      |
| suggestion_type  | varchar   |
| source_value     | varchar   |
| suggested_value  | varchar   |
| confidence_score | decimal   |
| status           | varchar   |
| created_at       | timestamp |

---

# Quan hệ

categories
1:N
products
(qua category_id)

---

products
1:N
click_logs
(qua product_id)

---

search_logs
1:N
click_logs
(qua search_log_id)

---

ai_suggestions
được tạo từ
search_logs
