# SwiftSearch Engine - Business Requirement Document

## 1. Project Overview

SwiftSearch Engine là hệ thống tìm kiếm sản phẩm dành cho nền tảng thương mại điện tử đa quốc gia.

Mục tiêu chính:

* Tăng tỷ lệ chuyển đổi tìm kiếm thành đơn hàng.
* Giảm tỷ lệ tìm kiếm không có kết quả.
* Hỗ trợ tìm kiếm đa ngôn ngữ.
* Cải thiện trải nghiệm người dùng thông qua autocomplete và ranking thông minh.
* Hỗ trợ hệ thống tự học từ hành vi tìm kiếm thực tế.

---

## 2. Business Goals

### BG-001 - Increase Search Conversion

Người dùng tìm thấy đúng sản phẩm nhanh hơn.

### BG-002 - Reduce Zero Result Search

Giảm số lượng truy vấn không trả về kết quả.

### BG-003 - Support Multi-language Search

Cho phép tìm kiếm giữa tiếng Việt, tiếng Anh và tiếng Thái.

### BG-004 - Improve Search Experience

Autocomplete, spellcheck và ranking phải hoạt động hiệu quả.

---

## 3. User Personas

### Buyer

Người tìm kiếm sản phẩm.

### Seller

Người đăng tải và quản lý sản phẩm.

### Admin

Người quản lý synonym, translation và AI suggestion.

---

## 4. Functional Requirements

### FR-001 Product Search

Tìm kiếm sản phẩm bằng từ khóa.

### FR-002 Autocomplete

Gợi ý từ khóa trong khi nhập.

### FR-003 Spellcheck

Tự động sửa lỗi chính tả phổ biến.

### FR-004 Synonym Expansion

Hỗ trợ từ đồng nghĩa.

### FR-005 Multilingual Search

Hỗ trợ tìm kiếm đa ngôn ngữ.

### FR-006 Ranking Engine

Xếp hạng kết quả theo mức độ liên quan.

### FR-007 Search Analytics

Thu thập dữ liệu tìm kiếm và hành vi người dùng.

### FR-008 AI Suggestion Engine

Đề xuất synonym và translation mới từ dữ liệu thực tế.

### FR-009 Admin Management

Quản trị synonym và AI suggestion.

### FR-010 AI Conversational Assistant

Trợ lý AI đàm thoại giúp quản trị viên tra cứu kho hàng và lập các đề xuất thay đổi từ điển bằng ngôn ngữ tự nhiên.

---

## 5. Non Functional Requirements

### NFR-001 Performance

Search latency < 50ms.

Autocomplete latency < 5ms.

### NFR-002 Scalability

Hỗ trợ tối thiểu 500 concurrent users.

### NFR-003 Availability

Search service hoạt động độc lập với hệ thống chính.

### NFR-004 Multi-tenancy

Hỗ trợ phân vùng dữ liệu theo tenant.

---

## 6. Scope

### In Scope

* Product Search
* Autocomplete
* Spellcheck
* Synonym
* Translation
* Ranking
* Analytics
* AI Suggestion
* AI Conversational Assistant

### Out of Scope

* Recommendation Engine
* Image Search
* Voice Search
* Personalized Search

---

## 7. Requirement Traceability

| Requirement | User Story     |
| ----------- | -------------- |
| FR-001      | US-002         |
| FR-002      | US-003         |
| FR-003      | US-004         |
| FR-004      | US-005         |
| FR-005      | US-006         |
| FR-006      | US-007         |
| FR-007      | US-008, US-009 |
| FR-008      | US-010         |
| FR-009      | US-011, US-012 |
| FR-010      | US-013         |

---

## 8. Business Rules (Quy tắc Nghiệp vụ)

Tài liệu này tập hợp tất cả các Quy tắc nghiệp vụ (Business Rules - BR) bất biến của hệ thống SwiftSearch Engine.

### A. Autocomplete (Gợi ý Từ khóa)
*   **BR-101**: Phải áp dụng kỹ thuật Debounce (150ms) ở Frontend để tránh spam request lên server khi người dùng gõ nhanh.
*   **BR-102**: Trường gợi ý phải bao gồm: Tên sản phẩm, Danh mục, và Thương hiệu.
*   **BR-103**: Không phân biệt chữ hoa, chữ thường khi thực hiện so khớp gợi ý.
*   **BR-104**: Tìm kiếm gợi ý có nhiều từ (multi-word suggest) phải khớp đồng thời tất cả các từ đơn (AND operator) trong chuỗi gợi ý để đảm bảo tính chính xác ngữ nghĩa và tránh hiển thị gợi ý lệch danh mục.

### B. Spellcheck (Sửa lỗi Chính tả)
*   **BR-201**: Từ điển sửa lỗi chính tả cục bộ phải được đồng bộ vào Redis Cache từ PostgreSQL của đúng tenant đó để tối ưu hóa hiệu năng tầng kiểm duyệt 1.
*   **BR-202**: Chỉ tự động sửa từ khóa nếu độ tin cậy của gợi ý (suggester confidence score) đạt trên ngưỡng quy định (ví dụ: `confidence > 0.8`).

### C. Synonym Expansion (Mở rộng Từ đồng nghĩa)
*   **BR-301**: Chỉ các từ đồng nghĩa có trạng thái `approved` (hoặc `active`) mới được sử dụng để mở rộng truy vấn.
*   **BR-302**: Tránh vòng lặp mở rộng vô hạn (infinite recursion) bằng cách chỉ mở rộng một cấp độ (không mở rộng bắc cầu: nếu A=B và B=C, tìm A chỉ mở rộng thành B, không mở rộng thành C trừ khi được khai báo trực tiếp).
*   **BR-303**: Từ điển đồng nghĩa của từng tenant được cache riêng biệt trong Redis để tránh xung đột dữ liệu.

### D. Multilingual Search (Tìm kiếm Đa ngôn ngữ)
*   **BR-401**: Trọng số tìm kiếm ngôn ngữ hiện tại của người dùng (ngôn ngữ UI) phải luôn được thiết lập cao nhất (boost: 5) để ưu tiên kết quả khớp ngôn ngữ gốc.
*   **BR-402**: Việc dịch thuật từ khóa tìm kiếm (Search-time) chỉ sử dụng từ điển dịch thuật tĩnh (`translations` table) được Admin quản trị hoặc kết quả dịch cache, tuyệt đối **không gọi Google Translate API thời gian thực** trong luồng search của người dùng để tránh nghẽn mạng và tăng độ trễ.
*   **BR-403**: Bộ nhớ đệm cache cho API Suggest phải được phân tách rõ rệt theo cả ngôn ngữ (`query:lang`) để tránh việc người dùng ở giao diện tiếng Anh nhận gợi ý cache của người dùng ở giao diện tiếng Việt.

### E. Ranking Engine (Xếp hạng Sản phẩm)
*   **BR-501**: Hệ số boost cho Featured Product và hệ số suy hao (decay) cho sản phẩm hết hàng phải được cấu hình động qua biến môi trường (`.env` / Environment Variables) để dễ dàng tinh chỉnh mà không cần sửa code.
*   **BR-502**: Điểm số BM25 gốc của từ khóa khớp chính xác (Exact Match) hoặc cụm từ (Phrase Match) phải luôn được ưu tiên hàng đầu trước khi áp dụng các bộ nhân trọng số nghiệp vụ.

### F. Click Tracking & Analytics (Theo dõi & Thống kê)
*   **BR-601**: Để đảm bảo tính tin cậy khi người dùng chuyển trang nhanh, Frontend có thể sử dụng API `navigator.sendBeacon` hoặc cấu hình Fetch ở chế độ `keepalive: true`.
*   **BR-602**: Nếu `search_log_id` truyền lên không hợp lệ hoặc không đúng định dạng UUID, API vẫn trả về `200 OK` nhưng ghi log warning ở backend để tránh làm gián đoạn trải nghiệm của người dùng khi có lỗi track log.
*   **BR-603**: Background Cron Job tổng hợp dữ liệu chạy hàng giờ để cập nhật số liệu CTR, Zero Result và phân bổ theo danh mục vào bảng tổng hợp.
*   **BR-604**: Background Cron Job dọn dẹp chạy lúc 2 AM hàng ngày để tự động xóa các dữ liệu raw logs (`search_logs`, `click_logs`) cũ hơn 90 ngày (Data Retention Policy) để tiết kiệm tài nguyên lưu trữ.
*   **BR-605**: Cung cấp API `POST /api/v1/admin/analytics/trigger?start_date=YYYY-MM-DD&end_date=YYYY-MM-DD` cho phép Admin chạy tổng hợp lại số liệu thủ công theo khoảng thời gian tùy chọn phục vụ việc sửa lỗi hoặc đồng bộ lại dữ liệu.

### G. AI Suggestion & Admin Dictionary Management (Quản trị Từ điển & Đề xuất AI)
*   **BR-701**: Phải cung cấp API thủ công để hỗ trợ trigger debug AI Suggestion trên môi trường local/development mà không bị phụ thuộc vào cron job.
*   **BR-702**: Không tạo đề xuất trùng lặp. Nếu một đề xuất (`source_value` + `suggested_value` + `suggestion_type`) đã tồn tại trong bảng `ai_suggestions` với trạng thái `pending` hoặc `approved`, hệ thống sẽ bỏ qua và không insert mới.
*   **BR-703**: Quyền truy cập: Chỉ các tài khoản có role `admin` hoặc `tenant_manager` mới được phép thao tác các API quản lý từ điển. Các API bắt buộc phải validate quyền.
*   **BR-704**: Đồng bộ Cache: Ngay khi bảng `synonyms` thay đổi (Insert/Update/Delete), hệ thống bắt buộc phải invalidate khóa cache tương ứng trong Redis (ví dụ khóa: `synonyms:{tenant_id}`).
*   **BR-705**: Chỉ Admin mới có quyền thực hiện thao tác duyệt/từ chối gợi ý AI.
*   **BR-706**: Các bản ghi bị từ chối (`rejected`) vẫn giữ lại trong bảng `ai_suggestions` để AI Worker tránh quét và đề xuất lại cùng một từ khóa trong tương lai.

### H. AI Assistant (Trợ lý AI Đàm thoại)
*   **BR-801**: Thao tác click Accept từ UI sẽ kích hoạt các REST API nghiệp vụ chuẩn, qua đó đảm bảo tính năng đồng bộ cache Redis tự động hoạt động hoàn chỉnh.
*   **BR-802**: Lịch sử chat được lưu trữ bền vững tại PostgreSQL dưới schema `search_svc` với 2 bảng `assistant_conversations` và `assistant_messages`, hỗ trợ liên kết khóa ngoại cascade delete khi Admin xóa phiên trò chuyện.
