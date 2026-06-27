import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { ShoppingBag, ArrowLeft, Layers, Star, Sparkles, Globe } from 'lucide-react';
import { useTenant } from '../context/TenantContext';
import { searchApi } from '../services/api';
import { Product } from '../types';
import Footer from '../components/Footer';

const ProductDetail: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { activeTenant } = useTenant();

  const [product, setProduct] = useState<Product | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [activeLang, setActiveLang] = useState<'vi' | 'en' | 'th'>('vi');

  useEffect(() => {
    const fetchProduct = async () => {
      if (!id) return;
      setLoading(true);
      setError(null);
      try {
        const data = await searchApi.getProduct(id, activeTenant.id);
        setProduct(data);
      } catch (err: any) {
        console.error(err);
        setError(err.message || 'Không thể tải thông tin sản phẩm.');
      } finally {
        setLoading(false);
      }
    };
    fetchProduct();
  }, [id, activeTenant.id]);

  // Helper to generate stars based on Product ID hashes for consistent visual quality
  const getMockRating = (productId: string) => {
    let hash = 0;
    for (let i = 0; i < productId.length; i++) {
      hash = productId.charCodeAt(i) + ((hash << 5) - hash);
    }
    const score = 3.5 + (Math.abs(hash) % 16) * 0.1;
    return Math.min(5, Math.round(score * 10) / 10);
  };

  if (loading) {
    return (
      <div className="app-container" style={{ minHeight: '100vh', display: 'flex', flexDirection: 'column' }}>
        <header className="header">
          <div className="header-logo" onClick={() => navigate('/')} style={{ cursor: 'pointer' }}>
            <ShoppingBag className="text-gradient" size={26} strokeWidth={2.5} />
            <span>Amaze<span style={{ color: 'var(--primary)' }}>Search</span></span>
          </div>
        </header>
        <div style={{ flex: 1, display: 'flex', justifyContent: 'center', alignItems: 'center' }}>
          <div style={{ fontSize: '1.2rem', fontWeight: 600, color: 'var(--text-secondary)' }}>
            Đang tải chi tiết sản phẩm...
          </div>
        </div>
      </div>
    );
  }

  if (error || !product) {
    return (
      <div className="app-container" style={{ minHeight: '100vh', display: 'flex', flexDirection: 'column' }}>
        <header className="header">
          <div className="header-logo" onClick={() => navigate('/')} style={{ cursor: 'pointer' }}>
            <ShoppingBag className="text-gradient" size={26} strokeWidth={2.5} />
            <span>Amaze<span style={{ color: 'var(--primary)' }}>Search</span></span>
          </div>
        </header>
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column', justifyContent: 'center', alignItems: 'center', gap: '16px' }}>
          <div style={{ fontSize: '1.2rem', fontWeight: 600, color: 'var(--danger, #ef4444)' }}>
            {error || 'Không tìm thấy sản phẩm.'}
          </div>
          <button onClick={() => navigate('/')} className="btn btn-primary" style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <ArrowLeft size={16} /> Quay về Trang chủ
          </button>
        </div>
      </div>
    );
  }

  const rating = getMockRating(product.id);
  const isOutOfStock = product.inventory <= 0;

  // Extract description translation mapping
  const descTranslated = activeLang === 'en' ? product.description_en : activeLang === 'th' ? product.description_th : product.description_vi;
  const nameTranslated = activeLang === 'en' ? product.product_name_en : activeLang === 'th' ? product.product_name_th : product.product_name_vi;

  return (
    <div className="app-container" style={{ minHeight: '100vh', display: 'flex', flexDirection: 'column', backgroundColor: '#f8fafc' }}>
      {/* HEADER SECTION */}
      <header className="header" style={{ borderBottom: '1px solid var(--border-color)', backgroundColor: '#ffffff' }}>
        <div className="header-logo" onClick={() => navigate('/')} style={{ cursor: 'pointer' }}>
          <ShoppingBag className="text-gradient" size={26} strokeWidth={2.5} />
          <span>Amaze<span style={{ color: 'var(--primary)' }}>Search</span></span>
        </div>

        <div className="header-actions">
          {/* Active Language Selector */}
          <div className="lang-selector" style={{ display: 'flex', gap: '12px' }}>
            <span 
              className={`lang-flag ${activeLang === 'vi' ? 'active' : ''}`}
              onClick={() => setActiveLang('vi')}
              title="Tiếng Việt (Gốc)"
              style={{ fontSize: '1.35rem', cursor: 'pointer', opacity: activeLang === 'vi' ? 1 : 0.4 }}
            >
              🇻🇳
            </span>
            <span 
              className={`lang-flag ${activeLang === 'en' ? 'active' : ''}`}
              onClick={() => setActiveLang('en')}
              title="Tiếng Anh (Dịch)"
              style={{ fontSize: '1.35rem', cursor: 'pointer', opacity: activeLang === 'en' ? 1 : 0.4 }}
            >
              🇺🇸
            </span>
            <span 
              className={`lang-flag ${activeLang === 'th' ? 'active' : ''}`}
              onClick={() => setActiveLang('th')}
              title="Tiếng Thái (Dịch)"
              style={{ fontSize: '1.35rem', cursor: 'pointer', opacity: activeLang === 'th' ? 1 : 0.4 }}
            >
              🇹🇭
            </span>
          </div>

          <button onClick={() => navigate('/admin')} className="btn btn-outline">
            Bảng Quản Trị
          </button>
        </div>
      </header>

      {/* PRODUCT DETAIL CONTAINER */}
      <main style={{ flex: 1, padding: '40px 20px', maxWidth: '1024px', width: '100%', margin: '0 auto' }}>
        {/* Back Link */}
        <button 
          onClick={() => navigate(-1)} 
          style={{ 
            display: 'inline-flex', 
            alignItems: 'center', 
            gap: '8px', 
            background: 'none', 
            border: 'none', 
            color: 'var(--text-secondary)', 
            cursor: 'pointer', 
            fontWeight: 600, 
            fontSize: '0.95rem',
            marginBottom: '24px',
            padding: '8px 0',
            transition: 'color 0.2s'
          }}
          onMouseEnter={(e) => (e.currentTarget.style.color = 'var(--primary)')}
          onMouseLeave={(e) => (e.currentTarget.style.color = 'var(--text-secondary)')}
        >
          <ArrowLeft size={18} />
          Quay lại trang trước
        </button>

        {/* Immersive Glassmorphic Card */}
        <div style={{
          backgroundColor: '#ffffff',
          borderRadius: '16px',
          boxShadow: '0 4px 30px rgba(0, 0, 0, 0.03), 0 1px 3px rgba(0, 0, 0, 0.02)',
          border: '1px solid var(--border-color)',
          display: 'flex',
          flexDirection: 'row',
          flexWrap: 'wrap',
          overflow: 'hidden'
        }}>
          {/* Left Column: Product Image */}
          <div style={{
            flex: '1 1 400px',
            position: 'relative',
            backgroundColor: '#f8fafc',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            minHeight: '400px',
            borderRight: '1px solid var(--border-color)'
          }}>
            {product.image_url ? (
              <img 
                src={product.image_url} 
                alt={product.product_name_vi}
                style={{
                  maxWidth: '100%',
                  maxHeight: '400px',
                  objectFit: 'contain',
                  padding: '24px'
                }}
                onError={(e) => {
                  (e.target as HTMLImageElement).style.display = 'none';
                }}
              />
            ) : null}

            {/* Fallback mockup box */}
            <div style={{
              position: 'absolute',
              top: 0,
              left: 0,
              right: 0,
              bottom: 0,
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              justifyContent: 'center',
              zIndex: 0,
              pointerEvents: 'none'
            }}>
              <Layers size={64} style={{ color: 'rgba(37, 99, 235, 0.08)', marginBottom: '12px' }} />
              <span style={{ fontSize: '0.85rem', color: 'var(--text-muted)', fontWeight: 600 }}>
                {product.brand || 'Amaze'}
              </span>
            </div>
          </div>

          {/* Right Column: Product Info & Details */}
          <div style={{
            flex: '1 1 500px',
            padding: '36px',
            display: 'flex',
            flexDirection: 'column'
          }}>
            {/* Top row: Brand & Stock Info */}
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px' }}>
              <span style={{ 
                fontSize: '0.85rem', 
                fontWeight: 700, 
                color: 'var(--primary)', 
                textTransform: 'uppercase', 
                letterSpacing: '0.05em',
                backgroundColor: 'rgba(37, 99, 235, 0.06)',
                padding: '4px 10px',
                borderRadius: '6px'
              }}>
                {product.brand || 'Amaze'}
              </span>
              <span style={{
                fontSize: '0.85rem',
                fontWeight: 600,
                color: isOutOfStock ? 'var(--danger, #ef4444)' : 'var(--success, #22c55e)',
                backgroundColor: isOutOfStock ? 'rgba(239, 68, 68, 0.06)' : 'rgba(34, 197, 94, 0.06)',
                padding: '4px 10px',
                borderRadius: '6px'
              }}>
                {isOutOfStock ? 'Hết hàng' : `Còn lại: ${product.inventory} chiếc`}
              </span>
            </div>

            {/* Title / Name */}
            <h1 style={{ 
              fontSize: '2rem', 
              fontWeight: 800, 
              color: 'var(--text-primary)', 
              lineHeight: '2.5rem',
              letterSpacing: '-0.02em',
              marginBottom: '8px'
            }}>
              {nameTranslated || product.product_name_vi}
            </h1>

            {/* Translation details if viewing non-original */}
            {activeLang !== 'vi' && (
              <div style={{ display: 'flex', alignItems: 'center', gap: '6px', color: 'var(--text-muted)', fontSize: '0.85rem', marginBottom: '16px' }}>
                <Globe size={14} />
                <span>Bản dịch tự động sang <strong>{activeLang === 'en' ? 'Tiếng Anh' : 'Tiếng Thái'}</strong></span>
              </div>
            )}

            {/* Rating section */}
            <div style={{ display: 'flex', alignItems: 'center', gap: '6px', marginBottom: '24px' }}>
              <div style={{ display: 'flex', color: '#fbbf24' }}>
                {Array.from({ length: 5 }).map((_, i) => (
                  <Star
                    key={i}
                    size={18}
                    fill={i < Math.floor(rating) ? 'currentColor' : 'none'}
                    style={{ opacity: i < Math.floor(rating) ? 1 : 0.2 }}
                  />
                ))}
              </div>
              <span style={{ fontSize: '0.9rem', color: 'var(--text-secondary)', fontWeight: 600, marginLeft: '4px' }}>
                {rating} / 5.0
              </span>
            </div>

            {/* Price section */}
            <div style={{ 
              backgroundColor: '#f8fafc', 
              padding: '20px 24px', 
              borderRadius: '12px',
              border: '1px solid var(--border-color)',
              marginBottom: '32px'
            }}>
              <span style={{ fontSize: '0.85rem', color: 'var(--text-muted)', fontWeight: 500, display: 'block', marginBottom: '4px' }}>
                Giá bán công khai:
              </span>
              <span style={{ fontSize: '2.25rem', fontWeight: 800, color: 'var(--primary)' }}>
                {product.price.toLocaleString('vi-VN')}đ
              </span>
            </div>

            {/* Description section */}
            <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', flex: 1 }}>
              <span style={{ 
                fontSize: '0.9rem', 
                fontWeight: 700, 
                color: 'var(--text-primary)', 
                display: 'flex', 
                alignItems: 'center', 
                gap: '6px'
              }}>
                <Sparkles size={16} style={{ color: 'var(--primary)' }} />
                Thông tin sản phẩm:
              </span>
              <p style={{ 
                fontSize: '0.975rem', 
                color: 'var(--text-secondary)', 
                lineHeight: '1.6rem',
                whiteSpace: 'pre-wrap'
              }}>
                {descTranslated || product.description_vi || 'Chưa có thông tin mô tả chi tiết cho sản phẩm này.'}
              </p>
            </div>
          </div>
        </div>
      </main>
      <Footer />
    </div>
  );
};

export default ProductDetail;
