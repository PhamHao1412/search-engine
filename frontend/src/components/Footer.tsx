import React from 'react';
import { ShoppingBag } from 'lucide-react';

const Footer: React.FC = () => {
  return (
    <footer className="footer">
      <div className="footer-content">
        <div className="footer-brand">
          <ShoppingBag className="text-gradient" size={20} strokeWidth={2.5} />
          <span>Swift<span style={{ color: 'var(--primary)' }}>Search</span></span>
        </div>
        <p className="footer-text">
          © {new Date().getFullYear()} SwiftSearch. Hệ thống tìm kiếm sản phẩm đa ngôn ngữ thời gian thực tối ưu bằng OpenSearch & AI.
        </p>
        <div className="footer-links">
          <a href="#" className="footer-link">Tài liệu API</a>
          <span className="footer-separator">•</span>
          <a href="#" className="footer-link">Chính sách bảo mật</a>
          <span className="footer-separator">•</span>
          <a href="#" className="footer-link">Hỗ trợ</a>
        </div>
      </div>
    </footer>
  );
};

export default Footer;
