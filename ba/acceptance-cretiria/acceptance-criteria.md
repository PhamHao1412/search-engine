# Acceptance Criteria - SwiftSearch Engine

Tài liệu này tổng hợp toàn bộ Tiêu chí chấp nhận (Acceptance Criteria - AC) cho các User Story thuộc dự án SwiftSearch Engine từ US-001 đến US-013.

---

## US-001 - Product Ingestion (Đồng bộ Sản phẩm)
*   **AC-1.1**: Dữ liệu thay đổi ở Product Service (thêm/sửa/xóa sản phẩm) phải được gửi sang Search Service thông qua message broker Apache Kafka dưới dạng sự kiện (Events).
*   **AC-1.2**: Khi nhận được sự kiện, Search Service phải tiến hành dịch tên sản phẩm và mô tả sản phẩm sang tiếng Anh và tiếng Thái (nếu ngôn ngữ gốc là tiếng Việt) và cập nhật chỉ mục OpenSearch trong vòng tối đa 2 giây.
*   **AC-1.3**: Hệ thống có cơ chế ghi nhận nhật ký đồng bộ (`sync_jobs`) và lưu trạng thái thất bại (`failed_translation`, `failed_opensearch`) để cho phép chạy lại (retry).

## US-002 - Product Search (Tìm kiếm Sản phẩm)
*   **AC-2.1**: Hệ thống phản hồi kết quả tìm kiếm với độ trễ (Search Latency) trung bình dưới 50ms cho các truy vấn thông thường.
*   **AC-2.2**: Người dùng có thể tìm kiếm sản phẩm bằng tên sản phẩm, thương hiệu hoặc mô tả. Dữ liệu trả về gồm ID, tên, thương hiệu, giá, tồn kho, ảnh, và trạng thái.
*   **AC-2.3**: Dữ liệu kết quả tìm kiếm phải được phân vùng độc lập theo từng Tenant ID (Multi-tenancy).

## US-003 - Autocomplete (Gợi ý Từ khóa)
*   **AC-3.1**: Hệ thống phản hồi danh sách từ khóa gợi ý khi người dùng gõ dở dang (ví dụ: gõ "bàn" gợi ý "bàn phím cơ", "bàn làm việc").
*   **AC-3.2**: Độ trễ API Autocomplete (Suggest API) phải dưới 5ms để đảm bảo trải nghiệm gõ phím mượt mà cho khách hàng.
*   **AC-3.3**: Gợi ý từ khóa sử dụng bộ phân tích Edge N-gram trên OpenSearch để tìm kiếm theo tiền tố khớp trực tiếp của từ gõ.

## US-004 - Spellcheck (Sửa lỗi Chính tả)
*   **AC-4.1**: Tự động phát hiện và sửa các lỗi chính tả phổ biến dựa trên từ điển cấu hình (ví dụ: gõ sai "ako" tự động sửa và tìm kiếm kết quả của "akko").
*   **AC-4.2**: Giao diện hiển thị thông báo rõ ràng cho người dùng biết hệ thống đã sửa lỗi: *"Đang hiển thị kết quả cho 'akko'. Tìm kiếm thay thế cho 'ako'"*.
*   **AC-4.3**: Hệ thống hỗ trợ sửa lỗi chính tả theo 2 cấp độ: Khớp tiền tố trực tiếp (Tier 1) và Khoảng cách chỉnh sửa Levenshtein Distance (Tier 2).

## US-005 - Synonym Expansion (Mở rộng Từ đồng nghĩa)
*   **AC-5.1**: Hỗ trợ thiết lập quy tắc từ đồng nghĩa một chiều (ví dụ: "smartphone" -> "iphone", gõ "smartphone" ra "iphone" nhưng gõ "iphone" không ra "smartphone").
*   **AC-5.2**: Hỗ trợ quy tắc từ đồng nghĩa hai chiều (ví dụ: "bàn phím" <-> "keyboard", gõ từ nào cũng hiển thị kết quả của cả hai).
*   **AC-5.3**: Việc đồng bộ cấu hình từ đồng nghĩa phải được áp dụng ngay lập tức lên bộ phân tích truy vấn (Query Analyzer) mà không cần lập chỉ mục lại toàn bộ sản phẩm.

## US-006 - Multilingual Search (Tìm kiếm Đa ngôn ngữ)
*   **AC-6.1**: Hệ thống tự động nhận diện ngôn ngữ của câu truy vấn (tiếng Việt, tiếng Anh hoặc tiếng Thái) bằng thư viện nhận diện ngôn ngữ.
*   **AC-6.2**: Khi phát hiện ngôn ngữ cụ thể, hệ thống sẽ ưu tiên tìm kiếm trên các trường ngôn ngữ tương ứng (ví dụ: `name_en` cho tiếng Anh, `name_th` cho tiếng Thái).
*   **AC-6.3**: Nếu không nhận diện được hoặc không tìm thấy kết quả trực tiếp, hệ thống tự động dịch từ khóa sang các ngôn ngữ khác để tìm kiếm mở rộng (Translation Fallback).

## US-007 - Ranking Engine (Công cụ Xếp hạng)
*   **AC-7.1**: Kết quả tìm kiếm được sắp xếp theo điểm số liên quan mặc định (BM25 Score của OpenSearch).
*   **AC-7.2**: Hệ thống tăng hạng hiển thị (Boost Score) cho các sản phẩm được đánh dấu nổi bật (`featured = true`).
*   **AC-7.3**: Hệ thống giảm điểm xếp hạng đối với các sản phẩm hết hàng hoặc tồn kho thấp (Inventory Decay Factor) để tối ưu tỷ lệ chuyển đổi.

## US-008 - Click Tracking (Ghi nhận Lượt click)
*   **AC-8.1**: Mỗi khi khách hàng click vào một sản phẩm trong trang kết quả tìm kiếm, hệ thống gửi sự kiện click về backend thông qua API `POST /api/v1/analytics/click`.
*   **AC-8.2**: Dữ liệu click phải chứa: `search_log_id` (để liên kết với lượt tìm kiếm), `product_id`, `tenant_id`, và `timestamp`.
*   **AC-8.3**: Ghi nhận bất đồng bộ lượt click để không chặn luồng chuyển trang hoặc tải ảnh của khách hàng.

## US-009 - Search Analytics (Báo cáo Phân tích)
*   **AC-9.1**: Admin có thể xem báo cáo tổng hợp hành vi tìm kiếm trên Admin Panel bao gồm: tổng lượt search, tổng lượt click, tỷ lệ click trung bình (CTR), số lượng truy vấn không có kết quả.
*   **AC-9.2**: Hiển thị bảng danh sách Top Từ khóa tìm kiếm nhiều nhất, Top Từ khóa tìm kiếm không có kết quả (Zero Result Queries), và danh sách các từ khóa có CTR thấp dưới ngưỡng (Low CTR Queries).
*   **AC-9.3**: Dữ liệu phân tích được tổng hợp định kỳ từ bảng nhật ký tìm kiếm của PostgreSQL.

## US-010 - AI Suggestion Engine (Đề xuất AI Tự động)
*   **AC-10.1**: Hệ thống tự động quét nhật ký tìm kiếm để lọc các từ khóa không ra kết quả hoặc CTR thấp.
*   **AC-10.2**: Sử dụng mô hình OpenAI để đối chiếu từ khóa đó với danh mục sản phẩm của Tenant và đề xuất các từ đồng nghĩa (synonyms) hoặc từ dịch thuật (translations) phù hợp.
*   **AC-10.3**: Đề xuất được lưu trữ ở trạng thái `pending` trong cơ sở dữ liệu và chờ Admin phê duyệt.

## US-011 - Synonym Management (Quản lý Từ điển Thủ công)
*   **AC-11.1**: Admin có thể tạo thủ công quy tắc từ đồng nghĩa (Synonyms) một chiều hoặc hai chiều và quy tắc sửa lỗi chính tả (Spellcheck) thông qua giao diện Admin Panel.
*   **AC-11.2**: Cung cấp tính năng xóa (Delete) các quy tắc từ điển cũ.
*   **AC-11.3**: Khi cấu hình từ điển thay đổi, hệ thống phải tự động xóa cache Redis liên quan để các lượt tìm kiếm tiếp theo nhận cấu hình mới.

## US-012 - Approve AI Suggestion (Duyệt Đề xuất AI)
*   **AC-12.1**: Admin xem danh sách các đề xuất tự động từ AI trong tab **Đề xuất AI**.
*   **AC-12.2**: Admin có thể click **Chấp nhận / Approve** (để áp dụng quy tắc từ điển và chuyển trạng thái đề xuất thành `approved`) hoặc **Từ chối / Reject** (chuyển trạng thái đề xuất thành `rejected`).
*   **AC-12.3**: Khi chấp nhận, dữ liệu được ghi nhận vào bảng cấu hình chính thức và tự động xóa cache Redis của Tenant.

## US-013 - Admin AI Assistant (Trợ lý AI Đàm thoại)
*   **AC-13.1**: Admin có thể chat trực tiếp với trợ lý AI bằng ngôn ngữ tự nhiên để tra cứu thông tin sản phẩm và đề xuất cấu hình từ điển.
*   **AC-13.2**: Toàn bộ lịch sử trò chuyện và danh sách cuộc hội thoại được lưu trữ bền vững tại PostgreSQL, hỗ trợ tính năng tạo hội thoại mới (New Chat) và xóa cuộc hội thoại.
*   **AC-13.3**: Mọi đề xuất thay đổi cấu hình ghi/xóa từ điển của AI được hiển thị dưới dạng **Action Card** cùng nút bấm Chấp nhận/Từ chối. Trạng thái của thẻ duyệt được lưu bền vững trong database.
*   **AC-13.4**: Trợ lý AI chỉ hoạt động trong phạm vi Tenant đang đăng nhập và không thể thực thi lệnh database trực tiếp mà phải thông qua cổng kiểm duyệt của Admin (Human-in-the-loop).
