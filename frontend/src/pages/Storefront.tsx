import React, { useState, useEffect, useRef } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { 
  Search, 
  SlidersHorizontal, 
  Star, 
  Info, 
  X, 
  Database, 
  ShoppingBag, 
  Layers, 
  ArrowRight,
  Sparkles,
  Zap
} from 'lucide-react';
import { useTenant } from '../context/TenantContext';
import { searchApi, getSearchDebugInfo } from '../services/api';
import { Product, SearchDebugInfo, Suggestion } from '../types';
import Footer from '../components/Footer';

const HOT_SUGGESTIONS: Record<string, string[]> = {
  'd3b07384-d113-4956-a5db-251d50c18d01': [
    'Bàn phím cơ Akko',
    'Chuột Logitech G Pro',
    'Bàn phím Ajazz',
    'Chuột không dây',
    'Keycap'
  ],
  'a1a2a3a4-b1b2-c1c2-d1d2-e1e2e3e4e5e6': [
    'Son dưỡng môi',
    'Son môi đỏ',
    'Mỹ phẩm dưỡng da',
    'Lip balm',
    'Kem chống nắng'
  ],
};

const Storefront: React.FC = () => {
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const urlQuery = searchParams.get('q') || '';
  const urlProductId = searchParams.get('productId') || '';
  const { activeTenant } = useTenant();

  // Search & Results States
  const [query, setQuery] = useState(urlQuery);
  const [products, setProducts] = useState<Product[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [searchLogId, setSearchLogId] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Suggestions States
  const [suggestions, setSuggestions] = useState<Suggestion[]>([]);
  const [showSuggestions, setShowSuggestions] = useState(false);
  const [activeIndex, setActiveIndex] = useState(-1);
  const dropdownRef = useRef<HTMLDivElement>(null);

  // Filter States
  const [selectedBrands, setSelectedBrands] = useState<string[]>([]);
  const [minPrice, setMinPrice] = useState('');
  const [maxPrice, setMaxPrice] = useState('');
  const [inStockOnly, setInStockOnly] = useState(false);

  // Available brands in current search results
  const [availableBrands, setAvailableBrands] = useState<string[]>([]);

  // Visual/UX States
  const [activeLang, setActiveLang] = useState<'vi' | 'en' | 'th'>('vi');
  const [selectedProduct, setSelectedProduct] = useState<{ product: Product; index: number } | null>(null);
  const [isDrawerOpen, setIsDrawerOpen] = useState(false);
  const [debugInfo, setDebugInfo] = useState<SearchDebugInfo | null>(null);
  const [spellcheckCorrected, setSpellcheckCorrected] = useState<string>('');
  const [autoCorrected, setAutoCorrected] = useState<boolean>(false);

  // Simulation state for Redis Cache
  const searchHistoryRef = useRef<Set<string>>(new Set());
  const prevTenantId = useRef(activeTenant.id);

  // Sync input value with URL parameter
  useEffect(() => {
    setQuery(urlQuery);
    setPage(1);
  }, [urlQuery]);

  // Trigger search when query parameter, page, or tenant changes
  useEffect(() => {
    handleSearch(urlQuery, page);
  }, [urlQuery, page, activeTenant.id]);

  // Debounce API call for autocomplete suggestions
  useEffect(() => {
    const cleanQuery = query.trim();
    if (cleanQuery.length < 2) {
      setSuggestions([]);
      return;
    }

    const handler = setTimeout(async () => {
      const results = await searchApi.getSuggestions(cleanQuery, activeTenant.id);
      setSuggestions(results);
    }, 150); // 150ms debounce delay as per BR-001

    return () => clearTimeout(handler);
  }, [query, activeTenant.id]);

  // Click outside to close dropdown suggestions list
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setShowSuggestions(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  // Extract unique brands whenever products load
  useEffect(() => {
    if (products.length > 0) {
      const brands = Array.from(
        new Set(products.map((p) => p.brand || 'Unbranded').filter(Boolean))
      );
      setAvailableBrands(brands);
    } else {
      setAvailableBrands([]);
    }
  }, [products]);

  // Auto-open product detail modal if productId is present in URL
  useEffect(() => {
    if (urlProductId && products.length > 0) {
      const idx = products.findIndex(p => p.id === urlProductId);
      if (idx !== -1) {
        setSelectedProduct({ product: products[idx], index: idx });
        
        // Track the click automatically
        const clickPayload = {
          search_log_id: searchLogId || '00000000-0000-0000-0000-000000000000',
          product_id: urlProductId,
          query: urlQuery || '(rỗng)',
          position: idx + 1,
        };
        searchApi.trackClick(clickPayload, activeTenant.id);

        // Remove the productId from query parameters so modal can be closed without reopening
        const newParams = new URLSearchParams(searchParams);
        newParams.delete('productId');
        setSearchParams(newParams, { replace: true });
      }
    }
  }, [urlProductId, products, searchLogId, urlQuery, activeTenant.id, searchParams, setSearchParams]);

  // Reset filters when tenant changes
  useEffect(() => {
    if (prevTenantId.current !== activeTenant.id) {
      prevTenantId.current = activeTenant.id;
      setQuery('');
      setSearchParams({});
      setSelectedBrands([]);
      setMinPrice('');
      setMaxPrice('');
      setInStockOnly(false);
      setPage(1);
    }
  }, [activeTenant.id, setSearchParams]);

  const handleSearch = async (searchTerm: string, targetPage: number) => {
    setLoading(true);
    setError(null);
    const startTime = performance.now();

    try {
      const cleanTerm = searchTerm.trim();
      const res = await searchApi.searchProducts(cleanTerm, targetPage, 20, activeTenant.id);
      
      const endTime = performance.now();
      const searchTimeMs = Math.round(endTime - startTime);

      setProducts(res.products || []);
      setTotal(res.total || 0);
      setTotalPages(res.total_pages || 1);
      setSearchLogId(res.search_log_id || '');
      setSpellcheckCorrected(res.spellcheck_corrected || '');
      setAutoCorrected(res.auto_corrected || false);

      // Simulate cache logic: if same query is searched within session, treat as Redis HIT
      const cacheKey = `${activeTenant.id}-${cleanTerm.toLowerCase()}-${targetPage}`;
      const isCached = searchHistoryRef.current.has(cacheKey);
      if (!isCached && cleanTerm) {
        searchHistoryRef.current.add(cacheKey);
      }

      // Generate debug panel metrics
      const debug = getSearchDebugInfo(cleanTerm || '(trống)', searchTimeMs, isCached && !!cleanTerm);
      if (res.spellcheck_corrected) {
        debug.spellcheck = res.spellcheck_corrected;
      } else {
        debug.spellcheck = '';
      }
      setDebugInfo(debug);
    } catch (err: any) {
      console.error(err);
      setError(err.message || 'Có lỗi xảy ra khi kết nối tới dịch vụ tìm kiếm.');
      setProducts([]);
      setTotal(0);
      setTotalPages(1);
    } finally {
      setLoading(false);
    }
  };

  const onSearchSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    setPage(1);
    setShowSuggestions(false);
    setSearchParams({ q: query.trim() });
  };

  const handleSuggestionClick = (suggestion: Suggestion) => {
    setQuery(suggestion.text);
    setShowSuggestions(false);
    navigate(`/products/${suggestion.id}`);
  };

  const handleHotSuggestionClick = (keyword: string) => {
    setQuery(keyword);
    setPage(1);
    setShowSuggestions(false);
    setSearchParams({ q: keyword.trim() });
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (!showSuggestions || suggestions.length === 0) return;

    if (e.key === 'ArrowDown') {
      e.preventDefault();
      setActiveIndex((prev) => (prev + 1) % suggestions.length);
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      setActiveIndex((prev) => (prev - 1 + suggestions.length) % suggestions.length);
    } else if (e.key === 'Enter') {
      if (activeIndex >= 0 && activeIndex < suggestions.length) {
        e.preventDefault();
        handleSuggestionClick(suggestions[activeIndex]);
      }
    } else if (e.key === 'Escape') {
      setShowSuggestions(false);
      setActiveIndex(-1);
    }
  };

  const handleProductClick = (product: Product, index: number) => {
    // Click tracking payload: non-blocking API call
    const clickPayload = {
      search_log_id: searchLogId || '00000000-0000-0000-0000-000000000000',
      product_id: product.id,
      query: urlQuery || '(rỗng)',
      position: index + 1, // 1-indexed position
    };
    
    searchApi.trackClick(clickPayload, activeTenant.id);
    navigate(`/products/${product.id}`);
  };

  // Helper to generate stars based on Product ID hashes for consistent visual quality
  const getMockRating = (id: string) => {
    let hash = 0;
    for (let i = 0; i < id.length; i++) {
      hash = id.charCodeAt(i) + ((hash << 5) - hash);
    }
    const score = 3.5 + (Math.abs(hash) % 16) * 0.1; // rating between 3.5 and 5.0
    return Math.min(5, Math.round(score * 10) / 10);
  };

  // Client-side filtering logic to complement OpenSearch results
  const filteredProducts = products.filter((product) => {
    // Brand filter
    if (selectedBrands.length > 0 && !selectedBrands.includes(product.brand || 'Unbranded')) {
      return false;
    }
    // Price min filter
    if (minPrice && product.price < parseFloat(minPrice)) {
      return false;
    }
    // Price max filter
    if (maxPrice && product.price > parseFloat(maxPrice)) {
      return false;
    }
    // Stock filter
    if (inStockOnly && product.inventory <= 0) {
      return false;
    }
    return true;
  });

  return (
    <div className="app-container">
      {/* HEADER SECTION */}
      <header className="header">
        <div className="header-logo" onClick={() => navigate('/')} style={{ cursor: 'pointer' }}>
          <ShoppingBag className="text-gradient" size={26} strokeWidth={2.5} />
          <span>Amaze<span style={{ color: 'var(--primary)' }}>Search</span></span>
        </div>

        <div className="header-actions">
          {/* Active Language Selector */}
          <div className="lang-selector">
            <span 
              className={`lang-flag ${activeLang === 'vi' ? 'active' : ''}`}
              onClick={() => setActiveLang('vi')}
              title="Tiếng Việt (Gốc)"
            >
              🇻🇳
            </span>
            <span 
              className={`lang-flag ${activeLang === 'en' ? 'active' : ''}`}
              onClick={() => setActiveLang('en')}
              title="Tiếng Anh (Dịch)"
            >
              🇺🇸
            </span>
            <span 
              className={`lang-flag ${activeLang === 'th' ? 'active' : ''}`}
              onClick={() => setActiveLang('th')}
              title="Tiếng Thái (Dịch)"
            >
              🇹🇭
            </span>
          </div>

          {/* Nav link to admin panel */}
          <button onClick={() => navigate('/admin')} className="btn btn-outline">
            Admin
            <ArrowRight size={16} />
          </button>
        </div>
      </header>

      <main className="main-content">
        {/* COMPACT SEARCH AREA */}
        <section className="search-centerpiece" style={{ padding: '24px 0 16px' }}>
          <div ref={dropdownRef} style={{ position: 'relative', width: '100%', maxWidth: '600px', margin: '0 auto 16px' }}>
            <form onSubmit={onSearchSubmit} className="search-box-wrapper" style={{ margin: 0 }}>
              <input
                type="text"
                placeholder={`Tìm kiếm sản phẩm trong ${activeTenant.name}...`}
                value={query}
                onChange={(e) => {
                  setQuery(e.target.value);
                  setShowSuggestions(true);
                  setActiveIndex(-1);
                }}
                onFocus={() => setShowSuggestions(true)}
                onKeyDown={handleKeyDown}
                className="search-input"
              />
              <button type="submit" className="search-icon-btn">
                <Search size={22} />
              </button>
            </form>

            {/* Suggestions list Dropdown */}
            {showSuggestions && suggestions.length > 0 && (
              <div className="autocomplete-dropdown" style={{
                position: 'absolute',
                top: '105%',
                left: 0,
                right: 0,
                backgroundColor: '#ffffff',
                border: '1px solid var(--border-color)',
                borderRadius: '12px',
                boxShadow: 'var(--shadow-lg), var(--shadow-premium)',
                zIndex: 1000,
                maxHeight: '300px',
                overflowY: 'auto',
                padding: '6px 0',
                display: 'flex',
                flexDirection: 'column'
              }}>
                {suggestions.map((suggestion, index) => {
                  const isOutOfStock = suggestion.inventory <= 0;
                  const translatedSubtitle = suggestion.product_name_en || '';

                  return (
                    <div
                      key={suggestion.id || index}
                      onClick={() => handleSuggestionClick(suggestion)}
                      className="autocomplete-row-item"
                      style={{
                        display: 'flex',
                        alignItems: 'center',
                        gap: '12px',
                        padding: '10px 16px',
                        cursor: 'pointer',
                        transition: 'background-color 0.15s ease',
                        backgroundColor: index === activeIndex ? 'var(--bg-tertiary)' : 'transparent',
                        borderBottom: index < suggestions.length - 1 ? '1px solid var(--bg-secondary)' : 'none'
                      }}
                      onMouseEnter={(e) => {
                        if (index !== activeIndex) {
                          e.currentTarget.style.backgroundColor = 'var(--bg-secondary)';
                        }
                      }}
                      onMouseLeave={(e) => {
                        if (index !== activeIndex) {
                          e.currentTarget.style.backgroundColor = 'transparent';
                        }
                      }}
                    >
                      {/* Left: Product Thumbnail */}
                      <div style={{ width: '40px', height: '40px', display: 'flex', alignItems: 'center', justifyContent: 'center', backgroundColor: 'var(--bg-tertiary)', borderRadius: '6px', overflow: 'hidden', flexShrink: 0, border: '1px solid var(--border-color)' }}>
                        {suggestion.image_url ? (
                          <img
                            src={suggestion.image_url}
                            alt=""
                            style={{ width: '100%', height: '100%', objectFit: 'cover' }}
                            onError={(e) => {
                              (e.target as HTMLImageElement).style.display = 'none';
                            }}
                          />
                        ) : (
                          <Search size={18} style={{ color: 'var(--text-muted)' }} />
                        )}
                      </div>

                      {/* Middle: Product info */}
                      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
                        <span style={{ fontSize: '0.9rem', fontWeight: 600, color: 'var(--text-primary)', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                          {suggestion.product_name_vi}
                        </span>
                        <span style={{ fontSize: '0.75rem', color: 'var(--text-secondary)', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                          {suggestion.brand || 'Chưa phân loại'} {translatedSubtitle ? `• ${translatedSubtitle}` : ''}
                        </span>
                      </div>

                      {/* Right: Price & Stock status */}
                      <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'flex-end', flexShrink: 0 }}>
                        <span style={{ fontSize: '0.85rem', fontWeight: 700, color: 'var(--primary)' }}>
                          {suggestion.price.toLocaleString('vi-VN')}đ
                        </span>
                        <span style={{ fontSize: '0.7rem', fontWeight: 500, color: isOutOfStock ? 'var(--danger)' : 'var(--success)' }}>
                          {isOutOfStock ? 'Hết hàng' : `Còn ${suggestion.inventory}`}
                        </span>
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </div>

          {/* Hot Suggestions */}
          <div className="search-suggestions" style={{ justifyContent: 'center' }}>
            <span style={{ display: 'flex', alignItems: 'center', gap: '4px', fontSize: '0.85rem', color: 'var(--text-muted)', fontWeight: 500, marginRight: '4px' }}>
              <Sparkles size={14} style={{ color: 'var(--warning)' }} /> Từ khóa hot:
            </span>
            {HOT_SUGGESTIONS[activeTenant.id]?.map((suggest) => (
              <button
                key={suggest}
                onClick={() => handleHotSuggestionClick(suggest)}
                className="suggestion-tag"
              >
                {suggest}
              </button>
            ))}
          </div>
        </section>

        {/* STOREFRONT GRID LAYOUT */}
        <div className="storefront-layout">
          {/* LEFT SIDEBAR FILTERS */}
          <aside className="filter-sidebar">
            <div className="filter-section">
              <div className="filter-title">
                <span>Bộ lọc Tìm kiếm</span>
                <SlidersHorizontal size={16} style={{ color: 'var(--text-muted)' }} />
              </div>
              <hr style={{ border: 'none', height: '1px', backgroundColor: 'var(--border-color)', margin: '12px 0 16px' }} />
            </div>

            {/* Brand Filter */}
            {availableBrands.length > 0 && (
              <div className="filter-section">
                <div className="filter-title">Thương hiệu</div>
                <div className="filter-options">
                  {availableBrands.map((brand) => (
                    <label key={brand} className="filter-checkbox-label">
                      <input
                        type="checkbox"
                        checked={selectedBrands.includes(brand)}
                        onChange={(e) => {
                          if (e.target.checked) {
                            setSelectedBrands([...selectedBrands, brand]);
                          } else {
                            setSelectedBrands(selectedBrands.filter((b) => b !== brand));
                          }
                        }}
                      />
                      {brand}
                    </label>
                  ))}
                </div>
              </div>
            )}

            {/* Price Filter */}
            <div className="filter-section">
              <div className="filter-title">Khoảng giá (VND)</div>
              <div className="price-inputs">
                <input
                  type="number"
                  placeholder="Min"
                  value={minPrice}
                  onChange={(e) => setMinPrice(e.target.value)}
                  className="price-input"
                />
                <span style={{ color: 'var(--text-muted)' }}>-</span>
                <input
                  type="number"
                  placeholder="Max"
                  value={maxPrice}
                  onChange={(e) => setMaxPrice(e.target.value)}
                  className="price-input"
                />
              </div>
            </div>

            {/* Stock Filter */}
            <div className="filter-section">
              <div className="filter-title">Trạng thái kho</div>
              <div className="filter-options">
                <label className="filter-checkbox-label">
                  <input
                    type="checkbox"
                    checked={inStockOnly}
                    onChange={(e) => setInStockOnly(e.target.checked)}
                  />
                  Chỉ hiện sản phẩm còn hàng
                </label>
              </div>
            </div>
          </aside>

          {/* MAIN PRODUCT LIST AREA */}
          <section className="product-area">
            {/* Header info */}
            <div className="product-results-header">
              <div>
                <span>Tìm thấy <strong>{total}</strong> kết quả (đã lọc: <strong>{filteredProducts.length}</strong>)</span>
                {urlQuery && <span> cho từ khóa "<strong>{urlQuery}</strong>"</span>}
              </div>
              <div style={{ fontSize: '0.85rem' }}>
                Tenant: <span style={{ color: 'var(--primary)', fontWeight: 600 }}>{activeTenant.name}</span>
              </div>
            </div>

            {/* Spellcheck Notice Banner */}
            {spellcheckCorrected && (
              <div 
                className="spellcheck-banner" 
                style={{
                  background: 'var(--card-bg)',
                  border: '1px solid var(--border-color)',
                  borderRadius: '8px',
                  padding: '12px 16px',
                  marginBottom: '16px',
                  display: 'flex',
                  alignItems: 'center',
                  gap: '8px',
                  fontSize: '0.9rem',
                  boxShadow: '0 2px 8px rgba(0,0,0,0.05)',
                  animation: 'fadeIn 0.3s ease',
                }}
              >
                <span style={{ color: 'var(--primary)', fontSize: '1.1rem' }}>✨</span>
                {autoCorrected ? (
                  <div>
                    Hiển thị kết quả cho <strong style={{ color: 'var(--primary)' }}>{spellcheckCorrected}</strong> (thay vì <em>"{urlQuery || ''}"</em>)
                  </div>
                ) : (
                  <div>
                    Có phải bạn muốn tìm:{' '}
                    <button
                      onClick={() => {
                        setSearchParams({ q: spellcheckCorrected });
                      }}
                      style={{
                        background: 'none',
                        border: 'none',
                        color: 'var(--primary)',
                        fontWeight: 'bold',
                        textDecoration: 'underline',
                        cursor: 'pointer',
                        padding: 0,
                        font: 'inherit',
                      }}
                    >
                      {spellcheckCorrected}
                    </button>
                    ?
                  </div>
                )}
              </div>
            )}

            {/* Loading / Error States */}
            {loading ? (
              <div className="empty-state">
                <div className="spinner" style={{ color: 'var(--primary)', marginBottom: '16px' }}>
                  <Database size={40} />
                </div>
                <div className="empty-state-text">Đang tải kết quả từ OpenSearch...</div>
              </div>
            ) : error ? (
              <div className="empty-state" style={{ border: '1px solid #fee2e2', borderRadius: '12px', background: '#fffbeb' }}>
                <span style={{ color: 'var(--danger)', fontSize: '2rem', marginBottom: '8px' }}>⚠️</span>
                <div className="empty-state-text" style={{ color: 'var(--text-primary)', fontWeight: 600 }}>Không thể hoàn thành tìm kiếm</div>
                <div style={{ color: 'var(--text-secondary)', fontSize: '0.85rem', marginTop: '4px' }}>{error}</div>
                <button 
                  onClick={() => handleSearch(urlQuery, page)} 
                  className="btn btn-primary" 
                  style={{ marginTop: '16px', padding: '8px 16px', fontSize: '0.85rem' }}
                >
                  Thử lại
                </button>
              </div>
            ) : filteredProducts.length === 0 ? (
              <div className="empty-state">
                <ShoppingBag className="empty-state-icon" size={40} style={{ color: 'var(--text-muted)' }} />
                <div className="empty-state-text">Không tìm thấy sản phẩm nào khớp với điều kiện lọc.</div>
                {products.length === 0 && (
                  <div style={{ fontSize: '0.8rem', color: 'var(--text-muted)', marginTop: '4px' }}>
                    Thử đổi từ khóa hoặc đồng bộ database ở trang quản trị.
                  </div>
                )}
              </div>
            ) : (
              /* Product Grid Display */
              <div className="product-grid">
                {filteredProducts.map((product, idx) => {
                  const rating = getMockRating(product.id);
                  const isOutOfStock = product.inventory <= 0;

                  // Selection of subtitle based on language switcher
                  let translatedSubtitle = '';
                  if (activeLang === 'en' && product.product_name_en) {
                    translatedSubtitle = product.product_name_en;
                  } else if (activeLang === 'th' && product.product_name_th) {
                    translatedSubtitle = product.product_name_th;
                  } else {
                    translatedSubtitle = product.product_name_en || product.product_name_th || 'Dịch thuật đang cập nhật...';
                  }

                  return (
                    <div
                      key={product.id}
                      onClick={() => handleProductClick(product, idx)}
                      className="product-card"
                    >
                      {product.featured && <span className="product-tag-featured">Nổi bật</span>}
                      
                      {/* Product Image Panel (Fallback implementation) */}
                      <div className="product-img-wrapper">
                        {product.image_url ? (
                          <img
                            src={product.image_url}
                            alt={product.product_name_vi}
                            className="product-img"
                            onError={(e) => {
                              // If loading image fails, show default placeholder
                              (e.target as HTMLImageElement).style.display = 'none';
                            }}
                          />
                        ) : null}
                        
                        {/* Default visual mockup placeholder for products */}
                        <div className="product-img-placeholder">
                          <Layers size={36} style={{ color: 'rgba(37, 99, 235, 0.15)' }} />
                          <span style={{ fontSize: '0.75rem', fontWeight: 500 }}>{product.brand || 'Amaze'}</span>
                        </div>
                      </div>

                      {/* Product Card Body */}
                      <div className="product-card-body">
                        <span className="product-brand">{product.brand || 'Chưa phân loại'}</span>
                        <h3 className="product-name-vi">{product.product_name_vi}</h3>
                        <p className="product-name-translated" title={translatedSubtitle}>
                          {translatedSubtitle}
                        </p>

                        <div className="rating-stars" style={{ marginBottom: '12px' }}>
                          {Array.from({ length: 5 }).map((_, i) => (
                            <Star
                              key={i}
                              size={14}
                              fill={i < Math.floor(rating) ? 'currentColor' : 'none'}
                              style={{ opacity: i < Math.floor(rating) ? 1 : 0.2 }}
                            />
                          ))}
                          <span style={{ fontSize: '0.75rem', color: 'var(--text-secondary)', marginLeft: '4px', fontWeight: 500 }}>
                            {rating}
                          </span>
                        </div>

                        <div className="product-card-footer">
                          <span className="product-price">
                            {product.price.toLocaleString('vi-VN')}đ
                          </span>
                          <span className={`product-stock ${isOutOfStock ? 'out-of-stock' : 'in-stock'}`}>
                            {isOutOfStock ? 'Hết hàng' : `Còn ${product.inventory}`}
                          </span>
                        </div>
                      </div>
                    </div>
                  );
                })}
              </div>
            )}

            {/* Pagination UI */}
            {totalPages > 1 && (
              <div className="pagination">
                <button
                  disabled={page === 1}
                  onClick={() => setPage(page - 1)}
                  className="btn btn-outline"
                  style={{ padding: '6px 12px', opacity: page === 1 ? 0.5 : 1 }}
                >
                  Trước
                </button>
                <span className="pagination-info">
                  Trang {page} / {totalPages}
                </span>
                <button
                  disabled={page === totalPages}
                  onClick={() => setPage(page + 1)}
                  className="btn btn-outline"
                  style={{ padding: '6px 12px', opacity: page === totalPages ? 0.5 : 1 }}
                >
                  Sau
                </button>
              </div>
            )}
          </section>
        </div>
      </main>

      {/* FLOATING ACTION BUTTON FOR SEARCH DEBUG SIDEBAR */}
      <div 
        onClick={() => setIsDrawerOpen(!isDrawerOpen)} 
        className="debug-toggle-btn"
        title="Bảng phân tích từ khóa tìm kiếm"
      >
        <Zap size={22} style={{ color: isDrawerOpen ? 'var(--warning)' : '#ffffff' }} />
      </div>

      {/* SEARCH DEBUG PANEL DRAWDER */}
      <aside className={`debug-drawer ${isDrawerOpen ? 'open' : ''}`}>
        <div className="debug-drawer-header">
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <Zap size={18} style={{ color: 'var(--primary)' }} />
            <h3 style={{ margin: 0 }}>Search Debug Panel</h3>
          </div>
          <button onClick={() => setIsDrawerOpen(false)} style={{ cursor: 'pointer', display: 'flex' }}>
            <X size={20} />
          </button>
        </div>

        <div className="debug-drawer-content">
          {debugInfo ? (
            <>
              {/* Original */}
              <div className="debug-metric-card">
                <div className="debug-metric-label">Original Query</div>
                <div className="debug-metric-value">"{debugInfo.originalQuery}"</div>
              </div>

              {/* Normalized */}
              <div className="debug-metric-card">
                <div className="debug-metric-label">Normalized</div>
                <div className="debug-metric-value" style={{ fontFamily: 'monospace' }}>
                  {debugInfo.normalized}
                </div>
              </div>

              {/* Spellcheck */}
              <div className="debug-metric-card">
                <div className="debug-metric-label">Spellcheck Suggestion</div>
                <div className="debug-metric-value">
                  {debugInfo.spellcheck ? (
                    <span style={{ color: 'var(--primary)', fontWeight: 600 }}>
                      {debugInfo.spellcheck}
                    </span>
                  ) : (
                    <span style={{ color: 'var(--text-muted)', fontStyle: 'italic' }}>
                      (Không phát hiện lỗi chính tả)
                    </span>
                  )}
                </div>
              </div>

              {/* Synonyms */}
              <div className="debug-metric-card">
                <div className="debug-metric-label">Synonyms Expanded</div>
                <div className="debug-metric-value">
                  {debugInfo.synonyms.length > 0 ? (
                    <div style={{ display: 'flex', flexWrap: 'wrap', gap: '4px', marginTop: '4px' }}>
                      {debugInfo.synonyms.map((syn) => (
                        <span 
                          key={syn} 
                          style={{ fontSize: '0.8rem', background: 'var(--bg-tertiary)', padding: '2px 8px', borderRadius: '4px' }}
                        >
                          {syn}
                        </span>
                      ))}
                    </div>
                  ) : (
                    <span style={{ color: 'var(--text-muted)', fontStyle: 'italic' }}>
                      (Không tìm thấy từ đồng nghĩa)
                    </span>
                  )}
                </div>
              </div>

              {/* Translations */}
              <div className="debug-metric-card">
                <div className="debug-metric-label">Translations Map</div>
                <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', marginTop: '6px' }}>
                  <div style={{ fontSize: '0.85rem' }}>
                    🇻🇳 <strong>VI:</strong> {debugInfo.translations.vi}
                  </div>
                  <div style={{ fontSize: '0.85rem' }}>
                    🇺🇸 <strong>EN:</strong> {debugInfo.translations.en}
                  </div>
                  <div style={{ fontSize: '0.85rem' }}>
                    🇹🇭 <strong>TH:</strong> {debugInfo.translations.th}
                  </div>
                </div>
              </div>

              {/* Cache status */}
              <div className="debug-metric-card" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <div className="debug-metric-label" style={{ margin: 0 }}>Cache Status (Redis)</div>
                <span className={`debug-metric-badge ${debugInfo.cacheStatus === 'HIT' ? 'hit' : 'miss'}`}>
                  {debugInfo.cacheStatus === 'HIT' ? 'HIT ⚡' : 'MISS ❌'}
                </span>
              </div>

              {/* Response timing */}
              <div className="debug-metric-card" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <div className="debug-metric-label" style={{ margin: 0 }}>Search Latency</div>
                <div className="debug-metric-value" style={{ fontWeight: 700, color: debugInfo.searchTime < 100 ? 'var(--success)' : 'var(--warning)' }}>
                  {debugInfo.searchTime} ms
                </div>
              </div>
            </>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', height: '100%', color: 'var(--text-muted)', textAlign: 'center' }}>
              <Info size={36} style={{ marginBottom: '12px' }} />
              <p style={{ fontSize: '0.9rem' }}>Hãy thử tìm kiếm sản phẩm để hiển thị phân tích xử lý dữ liệu.</p>
            </div>
          )}
        </div>
      </aside>

      {/* PRODUCT DETAIL VIEW MODAL */}
      {selectedProduct && (
        <div className="modal-overlay" onClick={() => setSelectedProduct(null)}>
          <div className="modal-container" onClick={(e) => e.stopPropagation()}>
            <button onClick={() => setSelectedProduct(null)} className="modal-close-btn">
              <X size={18} />
            </button>

            <div className="modal-grid">
              {/* Image Column */}
              <div className="modal-image-panel">
                {selectedProduct.product.image_url ? (
                  <img
                    src={selectedProduct.product.image_url}
                    alt={selectedProduct.product.product_name_vi}
                    style={{ width: '100%', height: '100%', objectFit: 'cover' }}
                  />
                ) : (
                  <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '8px', color: 'var(--text-muted)' }}>
                    <Layers size={52} style={{ color: 'rgba(37, 99, 235, 0.15)' }} />
                    <span style={{ fontSize: '0.85rem', fontWeight: 600 }}>{selectedProduct.product.brand || 'Amaze'}</span>
                  </div>
                )}
              </div>

              {/* Details Column */}
              <div className="modal-details-panel">
                <div className="modal-product-brand">{selectedProduct.product.brand || 'Unbranded'}</div>
                <h2 className="modal-product-name">{selectedProduct.product.product_name_vi}</h2>

                {/* Multilingual Names Display */}
                <div className="modal-product-translations">
                  <div className="modal-trans-item">
                    🇺🇸 <strong>Tiếng Anh:</strong> {selectedProduct.product.product_name_en || 'Chưa cập nhật'}
                  </div>
                  <div className="modal-trans-item">
                    🇹🇭 <strong>Tiếng Thái:</strong> {selectedProduct.product.product_name_th || 'Chưa cập nhật'}
                  </div>
                </div>

                <div className="modal-product-price">
                  {selectedProduct.product.price.toLocaleString('vi-VN')} VND
                </div>

                <div style={{ display: 'flex', gap: '16px', fontSize: '0.85rem', color: 'var(--text-secondary)' }}>
                  <div>
                    Trạng thái: <span style={{ fontWeight: 600, color: selectedProduct.product.inventory > 0 ? 'var(--success)' : 'var(--danger)' }}>
                      {selectedProduct.product.inventory > 0 ? 'Còn hàng' : 'Hết hàng'}
                    </span>
                  </div>
                  <div>|</div>
                  <div>Số lượng trong kho: <strong>{selectedProduct.product.inventory}</strong></div>
                </div>
                
                {/* Simulated Description translations display */}
                <div style={{ marginTop: '24px', fontSize: '0.85rem', borderTop: '1px solid var(--border-color)', paddingTop: '16px', color: 'var(--text-secondary)' }}>
                  <div style={{ fontWeight: 600, color: 'var(--text-primary)', marginBottom: '4px' }}>Mô tả sản phẩm:</div>
                  <p style={{ lineHeight: '1.4', fontStyle: 'normal' }}>
                    {selectedProduct.product.description_vi || 'Sản phẩm chưa cập nhật mô tả chi tiết bằng tiếng Việt.'}
                  </p>
                  {selectedProduct.product.description_en && (
                    <p style={{ lineHeight: '1.4', fontStyle: 'italic', marginTop: '8px', fontSize: '0.8rem', color: 'var(--text-muted)' }}>
                      En description: {selectedProduct.product.description_en}
                    </p>
                  )}
                </div>
              </div>
            </div>
          </div>
        </div>
      )}
      <Footer />
    </div>
  );
};

export default Storefront;
