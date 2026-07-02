import React, { useState, useEffect, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import { Search, ShoppingBag, ArrowRight, Sparkles } from 'lucide-react';
import { useTenant } from '../context/TenantContext';
import { searchApi } from '../services/api';
import { Suggestion } from '../types';
import Footer from '../components/Footer';

// Dynamic hot keywords are loaded from search logs API

const Home: React.FC = () => {
  const navigate = useNavigate();
  const { activeTenant } = useTenant();
  const [activeLang, setActiveLang] = useState<'vi' | 'en' | 'th'>(
    (localStorage.getItem('swiftsearch_search_lang') as 'vi' | 'en' | 'th') || 'vi'
  );
  const [query, setQuery] = useState('');
  const [suggestions, setSuggestions] = useState<Suggestion[]>([]);
  const [showSuggestions, setShowSuggestions] = useState(false);
  const [activeIndex, setActiveIndex] = useState(-1);
  const [hotKeywords, setHotKeywords] = useState<string[]>([]);
  const dropdownRef = useRef<HTMLDivElement>(null);

  // Load real-time hot keywords
  useEffect(() => {
    const loadHotKeywords = async () => {
      const keywords = await searchApi.getHotKeywords(activeTenant.id);
      setHotKeywords(keywords);
    };
    loadHotKeywords();
  }, [activeTenant.id, activeLang]);

  const handleLangChange = (lang: 'vi' | 'en' | 'th') => {
    setActiveLang(lang);
    localStorage.setItem('swiftsearch_search_lang', lang);
  };

  // Debounced API call for Autocomplete suggestions
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

  const handleSearchSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    triggerSearch(query);
  };

  const triggerSearch = (searchTerm: string) => {
    const cleanTerm = searchTerm.trim();
    if (cleanTerm) {
      navigate(`/search?q=${encodeURIComponent(cleanTerm)}`);
    }
  };

  const handleSuggestionClick = (suggestion: Suggestion) => {
    setQuery(suggestion.text);
    setShowSuggestions(false);
    navigate(`/products/${suggestion.id}`);
  };

  const handleHotSuggestionClick = (keyword: string) => {
    setQuery(keyword);
    setShowSuggestions(false);
    triggerSearch(keyword);
  };

  // Keyboard navigation for Autocomplete dropdown menu list
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

  return (
    <div className="app-container" style={{ minHeight: '100vh', display: 'flex', flexDirection: 'column' }}>
      {/* HEADER */}
      <header className="header">
        <div className="header-logo" onClick={() => navigate('/')} style={{ cursor: 'pointer' }}>
          <ShoppingBag className="text-gradient" size={26} strokeWidth={2.5} />
          <span>Swift<span style={{ color: 'var(--primary)' }}>Search</span></span>
        </div>

        <div className="header-actions">
          {/* Active Language Selector */}
          <div className="lang-selector">
            <span 
              className={`lang-flag ${activeLang === 'vi' ? 'active' : ''}`}
              onClick={() => handleLangChange('vi')}
              title="Tiếng Việt (Gốc)"
            >
              🇻🇳
            </span>
            <span 
              className={`lang-flag ${activeLang === 'en' ? 'active' : ''}`}
              onClick={() => handleLangChange('en')}
              title="Tiếng Anh (Dịch)"
            >
              🇺🇸
            </span>
            <span 
              className={`lang-flag ${activeLang === 'th' ? 'active' : ''}`}
              onClick={() => handleLangChange('th')}
              title="Tiếng Thái (Dịch)"
            >
              🇹🇭
            </span>
          </div>

          <button onClick={() => navigate('/admin')} className="btn btn-outline">
            Admin
            <ArrowRight size={16} />
          </button>
        </div>
      </header>

      {/* CENTERPIECE SEARCH CONTENT */}
      <main style={{ flex: 1, display: 'flex', flexDirection: 'column', justifyContent: 'flex-start', alignItems: 'center', padding: '16px 20px 80px' }}>
        <div style={{ maxWidth: '640px', width: '100%', display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
          
          <ShoppingBag size={32} style={{ color: 'var(--primary)', marginBottom: '8px', opacity: 0.9 }} />
          
          <h1 style={{ fontWeight: 800, fontSize: '2.5rem', marginBottom: '8px', textAlign: 'center', letterSpacing: '-0.03em', color: 'var(--text-primary)' }}>
            Bạn muốn tìm kiếm gì hôm nay?
          </h1>
          <p style={{ color: 'var(--text-secondary)', marginBottom: '16px', textAlign: 'center', fontSize: '1rem' }}>
            Hệ thống tìm kiếm thông tin sản phẩm tức thời cho cửa hàng <strong style={{ color: 'var(--primary)' }}>{activeTenant.name}</strong>.
          </p>

          {/* Autocomplete Input Container */}
          <div ref={dropdownRef} style={{ position: 'relative', width: '100%', marginBottom: '20px' }}>
            <form onSubmit={handleSearchSubmit} className="search-box-wrapper" style={{ margin: 0 }}>
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
                style={{ fontSize: '1.05rem', padding: '16px 20px' }}
              />
              <button type="submit" className="search-icon-btn" style={{ padding: '0 24px' }}>
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
                          {activeLang === 'en' && suggestion.product_name_en 
                            ? suggestion.product_name_en 
                            : activeLang === 'th' && suggestion.product_name_th 
                              ? suggestion.product_name_th 
                              : suggestion.product_name_vi}
                        </span>
                        <span style={{ fontSize: '0.75rem', color: 'var(--text-secondary)', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                          {suggestion.brand || 'Chưa phân loại'} {(() => {
                            const desc = activeLang === 'en' && suggestion.description_en 
                              ? suggestion.description_en 
                              : activeLang === 'th' && suggestion.description_th 
                                ? suggestion.description_th 
                                : suggestion.description_vi;
                            return desc ? `• ${desc}` : '';
                          })()}
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

          {/* Hot trending suggestions below search bar */}
          <div className="search-suggestions" style={{ flexWrap: 'wrap', justifyContent: 'center' }}>
            <span style={{ display: 'flex', alignItems: 'center', gap: '4px', fontSize: '0.85rem', color: 'var(--text-muted)', fontWeight: 500 }}>
              <Sparkles size={14} style={{ color: 'var(--warning)' }} /> Từ khóa hot:
            </span>
            {hotKeywords.map((suggest) => (
              <button
                key={suggest}
                onClick={() => handleHotSuggestionClick(suggest)}
                className="suggestion-tag"
                style={{ padding: '8px 14px', borderRadius: '20px' }}
              >
                {suggest}
              </button>
            ))}
          </div>

        </div>
      </main>
      <Footer />
    </div>
  );
};

export default Home;
