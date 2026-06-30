import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { 
  ArrowLeft, 
  RefreshCw, 
  Server, 
  Settings, 
  BookOpen, 
  Clock, 
  Sparkles, 
  CheckCircle2, 
  AlertTriangle,
  LayoutDashboard,
  ChevronLeft,
  ChevronRight,
  Search,
  TrendingUp,
  AlertCircle,
  HelpCircle,
  Plus,
  X
} from 'lucide-react';
import { useTenant } from '../context/TenantContext';
import { searchApi, adminApi, AISuggestion } from '../services/api';
import Footer from '../components/Footer';

interface SyncLog {
  timestamp: string;
  tenantId: string;
  tenantName: string;
  syncedCount: number;
  status: 'SUCCESS' | 'FAILED';
  errorMessage?: string;
}

const Admin: React.FC = () => {
  const navigate = useNavigate();
  const { activeTenant, setActiveTenantById, tenants } = useTenant();

  // Navigation & Layout states
  const [activeTab, setActiveTab] = useState<'dashboard' | 'ai-suggestions' | 'dictionaries' | 'sync'>('ai-suggestions');
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);

  // Sync state
  const [syncing, setSyncing] = useState(false);
  const [syncMessage, setSyncMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);
  const [syncLogs, setSyncLogs] = useState<SyncLog[]>([]);

  // AI Suggestions lists and total counts from Backend (independent for Typo and Synonym tables)
  const [typoSuggestions, setTypoSuggestions] = useState<AISuggestion[]>([]);
  const [synonymSuggestions, setSynonymSuggestions] = useState<AISuggestion[]>([]);
  const [typoTotal, setTypoTotal] = useState(0);
  const [synonymTotal, setSynonymTotal] = useState(0);
  const [loadingTypos, setLoadingTypos] = useState(false);
  const [loadingSynonyms, setLoadingSynonyms] = useState(false);

  // Dictionaries lists
  const [spellcheckRules, setSpellcheckRules] = useState<Array<{ id: string; typo_word: string; correct_word: string; status: string }>>([]);
  const [synonymRules, setSynonymRules] = useState<Array<{ id: string; keyword: string; synonym: string; status: string }>>([]);
  const [loadingDictionaries, setLoadingDictionaries] = useState(false);

  // Independent search & pagination states mapping directly to Backend params
  const [typoStatus, setTypoStatus] = useState<'pending' | 'approved' | 'rejected'>('pending');
  const [synonymStatus, setSynonymStatus] = useState<'pending' | 'approved' | 'rejected'>('pending');
  const [typoSearch, setTypoSearch] = useState('');
  const [synonymSearch, setSynonymSearch] = useState('');
  const [typoPage, setTypoPage] = useState(1);
  const [synonymPage, setSynonymPage] = useState(1);
  const itemsPerPage = 5;

  // Create Manual Dictionary Rules states
  const [showAddRuleModal, setShowAddRuleModal] = useState(false);
  const [newRuleType, setNewRuleType] = useState<'synonym' | 'spellcheck'>('synonym');
  const [newKeyword, setNewKeyword] = useState('');
  const [newSynonym, setNewSynonym] = useState('');
  const [isBidirectional, setIsBidirectional] = useState(false);
  const [newTypoWord, setNewTypoWord] = useState('');
  const [newCorrectWord, setNewCorrectWord] = useState('');
  const [actionLoading, setActionLoading] = useState(false);

  const handleAddRule = async (e: React.FormEvent) => {
    e.preventDefault();
    setActionLoading(true);
    try {
      if (newRuleType === 'synonym') {
        if (!newKeyword.trim() || !newSynonym.trim()) {
          alert('Vui lòng nhập đầy đủ Từ khóa chính và Từ đồng nghĩa');
          setActionLoading(false);
          return;
        }
        await adminApi.addSynonym(activeTenant.id, newKeyword.trim(), newSynonym.trim(), isBidirectional);
      } else {
        if (!newTypoWord.trim() || !newCorrectWord.trim()) {
          alert('Vui lòng nhập đầy đủ Từ gõ sai và Từ sửa đúng');
          setActionLoading(false);
          return;
        }
        await adminApi.addSpellcheck(activeTenant.id, newTypoWord.trim(), newCorrectWord.trim());
      }
      
      // Reset form states
      setNewKeyword('');
      setNewSynonym('');
      setIsBidirectional(false);
      setNewTypoWord('');
      setNewCorrectWord('');
      setShowAddRuleModal(false);

      // Refresh dictionaries list and search suggestions
      fetchDictionaries();
      fetchTypoSuggestions();
      fetchSynonymSuggestions();
    } catch (err: any) {
      alert(`Thêm quy tắc thất bại: ${err.message || 'Lỗi hệ thống'}`);
    } finally {
      setActionLoading(false);
    }
  };

  const handleDeleteSynonym = async (id: string) => {
    if (!window.confirm('Bạn có chắc chắn muốn xóa quy tắc từ đồng nghĩa này?')) return;
    try {
      await adminApi.deleteSynonym(activeTenant.id, id);
      fetchDictionaries();
      fetchSynonymSuggestions();
    } catch (err: any) {
      alert(`Xóa thất bại: ${err.message || 'Lỗi hệ thống'}`);
    }
  };

  const handleDeleteSpellcheck = async (id: string) => {
    if (!window.confirm('Bạn có chắc chắn muốn xóa quy tắc sửa chính tả này?')) return;
    try {
      await adminApi.deleteSpellcheck(activeTenant.id, id);
      fetchDictionaries();
      fetchTypoSuggestions();
    } catch (err: any) {
      alert(`Xóa thất bại: ${err.message || 'Lỗi hệ thống'}`);
    }
  };

  // Load sync logs from localStorage on mount
  useEffect(() => {
    const savedLogs = localStorage.getItem('search_sync_logs');
    if (savedLogs) {
      try {
        setSyncLogs(JSON.parse(savedLogs));
      } catch (e) {
        console.error('Failed to parse sync logs', e);
      }
    }
  }, []);

  const fetchTypoSuggestions = async () => {
    setLoadingTypos(true);
    try {
      const res = await adminApi.getAISuggestions(
        activeTenant.id,
        typoStatus,
        'typo',
        typoSearch,
        typoPage,
        itemsPerPage
      );
      setTypoSuggestions(res.suggestions);
      setTypoTotal(res.total);
    } catch (e) {
      console.error('Failed to fetch Typo suggestions', e);
    } finally {
      setLoadingTypos(false);
    }
  };

  const fetchSynonymSuggestions = async () => {
    setLoadingSynonyms(true);
    try {
      const res = await adminApi.getAISuggestions(
        activeTenant.id,
        synonymStatus,
        'synonym',
        synonymSearch,
        synonymPage,
        itemsPerPage
      );
      setSynonymSuggestions(res.suggestions);
      setSynonymTotal(res.total);
    } catch (e) {
      console.error('Failed to fetch Synonym suggestions', e);
    } finally {
      setLoadingSynonyms(false);
    }
  };

  const fetchDictionaries = async () => {
    setLoadingDictionaries(true);
    try {
      const spRules = await adminApi.getSpellcheckRules(activeTenant.id);
      setSpellcheckRules(spRules);
      
      const synRules = await adminApi.getSearchSynonyms(activeTenant.id);
      setSynonymRules(synRules);
    } catch (e) {
      console.error('Failed to fetch dictionary rules', e);
    } finally {
      setLoadingDictionaries(false);
    }
  };

  // Fetch data on active tenant, status, search keyword, or page index changes (Backend-driven)
  useEffect(() => {
    fetchTypoSuggestions();
  }, [activeTenant.id, typoStatus, typoSearch, typoPage]);

  useEffect(() => {
    fetchSynonymSuggestions();
  }, [activeTenant.id, synonymStatus, synonymSearch, synonymPage]);

  useEffect(() => {
    fetchDictionaries();
  }, [activeTenant.id]);

  const handleApproveSuggestion = async (id: string, type: 'typo' | 'synonym') => {
    try {
      await adminApi.approveAISuggestion(id, activeTenant.id);
      if (type === 'typo') {
        fetchTypoSuggestions();
      } else {
        fetchSynonymSuggestions();
      }
      fetchDictionaries();
    } catch (e: any) {
      alert(`Phê duyệt thất bại: ${e.message || 'Lỗi hệ thống'}`);
    }
  };

  const handleRejectSuggestion = async (id: string, type: 'typo' | 'synonym') => {
    try {
      await adminApi.rejectAISuggestion(id, activeTenant.id);
      if (type === 'typo') {
        fetchTypoSuggestions();
      } else {
        fetchSynonymSuggestions();
      }
    } catch (e: any) {
      alert(`Từ chối thất bại: ${e.message || 'Lỗi hệ thống'}`);
    }
  };

  const [generatingSuggestions, setGeneratingSuggestions] = useState(false);

  const handleGenerateAISuggestions = async () => {
    if (generatingSuggestions) return;
    setGeneratingSuggestions(true);
    try {
      await adminApi.generateAISuggestions(activeTenant.id);
      // Reset tables page pointers and trigger reload
      setTypoPage(1);
      setSynonymPage(1);
      await Promise.all([fetchTypoSuggestions(), fetchSynonymSuggestions()]);
    } catch (e: any) {
      alert(`Phân tích AI thất bại: ${e.message || 'Lỗi hệ thống'}`);
    } finally {
      setGeneratingSuggestions(false);
    }
  };

  const handleSyncDatabase = async () => {
    setSyncing(true);
    setSyncMessage(null);
    const timestamp = new Date().toLocaleString('vi-VN');

    try {
      const res = await searchApi.syncDatabase(activeTenant.id);
      
      const newLog: SyncLog = {
        timestamp,
        tenantId: activeTenant.id,
        tenantName: activeTenant.name,
        syncedCount: res.synced_count,
        status: 'SUCCESS',
      };

      const updatedLogs = [newLog, ...syncLogs].slice(0, 10);
      setSyncLogs(updatedLogs);
      localStorage.setItem('search_sync_logs', JSON.stringify(updatedLogs));

      setSyncMessage({
        type: 'success',
        text: `Đồng bộ hoàn tất! Đã cập nhật ${res.synced_count} sản phẩm lên OpenSearch.`,
      });
    } catch (err: any) {
      console.error(err);
      const newLog: SyncLog = {
        timestamp,
        tenantId: activeTenant.id,
        tenantName: activeTenant.name,
        syncedCount: 0,
        status: 'FAILED',
        errorMessage: err.message || 'Lỗi kết nối API',
      };

      const updatedLogs = [newLog, ...syncLogs].slice(0, 10);
      setSyncLogs(updatedLogs);
      localStorage.setItem('search_sync_logs', JSON.stringify(updatedLogs));

      setSyncMessage({
        type: 'error',
        text: `Đồng bộ thất bại: ${err.message || 'Không thể kết nối đến Backend Go.'}`,
      });
    } finally {
      setSyncing(false);
    }
  };

  const clearSyncLogs = () => {
    setSyncLogs([]);
    localStorage.removeItem('search_sync_logs');
  };

  // Pagination stats calculation
  const totalTypoPages = Math.ceil(typoTotal / itemsPerPage) || 1;
  const typoStartIndex = (typoPage - 1) * itemsPerPage;

  const totalSynonymPages = Math.ceil(synonymTotal / itemsPerPage) || 1;
  const synonymStartIndex = (synonymPage - 1) * itemsPerPage;

  return (
    <div className="admin-layout">
      {/* COLLAPSIBLE SIDEBAR */}
      <aside className={`admin-sidebar ${sidebarCollapsed ? 'collapsed' : ''}`}>
        <div className="admin-sidebar-header">
          <Settings className="text-gradient" size={26} strokeWidth={2.5} style={{ flexShrink: 0 }} />
          <span style={{ fontWeight: 800, fontSize: '1.25rem', letterSpacing: '-0.02em' }} className="admin-menu-text">
            Amaze<span style={{ color: 'var(--primary)' }}>Search</span>
          </span>
        </div>

        <button 
          onClick={() => setSidebarCollapsed(!sidebarCollapsed)}
          className="admin-sidebar-toggle"
          title={sidebarCollapsed ? "Mở rộng menu" : "Thu gọn menu"}
        >
          {sidebarCollapsed ? <ChevronRight size={14} /> : <ChevronLeft size={14} />}
        </button>

        <nav className="admin-sidebar-menu">
          <button 
            onClick={() => setActiveTab('dashboard')} 
            className={`admin-menu-item ${activeTab === 'dashboard' ? 'active' : ''}`}
          >
            <LayoutDashboard size={20} />
            <span className="admin-menu-text">Dashboard</span>
          </button>
          
          <button 
            onClick={() => setActiveTab('ai-suggestions')} 
            className={`admin-menu-item ${activeTab === 'ai-suggestions' ? 'active' : ''}`}
          >
            <Sparkles size={20} />
            <span className="admin-menu-text">Đề xuất từ AI</span>
          </button>

          <button 
            onClick={() => setActiveTab('dictionaries')} 
            className={`admin-menu-item ${activeTab === 'dictionaries' ? 'active' : ''}`}
          >
            <BookOpen size={20} />
            <span className="admin-menu-text">Cấu hình Từ khóa</span>
          </button>

          <button 
            onClick={() => setActiveTab('sync')} 
            className={`admin-menu-item ${activeTab === 'sync' ? 'active' : ''}`}
          >
            <RefreshCw size={20} />
            <span className="admin-menu-text">Đồng bộ dữ liệu</span>
          </button>
        </nav>

        <div style={{ marginTop: 'auto', padding: '16px' }} className="admin-menu-text">
          <button 
            onClick={() => navigate('/')} 
            className="btn btn-outline" 
            style={{ width: '100%', display: 'flex', gap: '8px', justifyContent: 'center' }}
          >
            <ArrowLeft size={14} />
            Cửa hàng
          </button>
        </div>
      </aside>

      {/* MAIN CONTAINER */}
      <div className="admin-main">
        {/* TOPBAR */}
        <header className="admin-topbar">
          <div>
            <h2 style={{ fontSize: '1.25rem', fontWeight: 800, color: 'var(--text-primary)' }}>
              {activeTab === 'dashboard' && 'Dashboard Analytics'}
              {activeTab === 'ai-suggestions' && 'AI Suggestion Engine'}
              {activeTab === 'dictionaries' && 'Active Search Dictionaries'}
              {activeTab === 'sync' && 'Database Sync Controls'}
            </h2>
          </div>

          {/* Tenant Switcher in Topbar */}
          <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
            <Server size={16} style={{ color: 'var(--text-muted)' }} />
            <span style={{ fontSize: '0.85rem', fontWeight: 600, color: 'var(--text-secondary)' }}>
              Cửa hàng hoạt động:
            </span>
            <select 
              value={activeTenant.id}
              onChange={(e) => setActiveTenantById(e.target.value)}
              className="btn btn-outline"
              style={{ padding: '6px 12px', height: '36px', fontSize: '0.85rem', fontWeight: 600, border: '1.5px solid var(--border-color)', borderRadius: 'var(--radius-sm)' }}
            >
              {tenants.map(t => (
                <option key={t.id} value={t.id}>{t.name}</option>
              ))}
            </select>
          </div>
        </header>

        {/* BODY */}
        <main className="admin-body">
          {/* TAB 1: DASHBOARD (MOCK/PLACEHOLDER FOR US-009) */}
          {activeTab === 'dashboard' && (
            <div>
              {/* Premium metrics grid */}
              <div className="metrics-grid">
                <div className="metric-card">
                  <div className="metric-card-bg-gradient" />
                  <div className="metric-card-header">
                    <span className="metric-card-title">Tổng số tìm kiếm</span>
                    <div className="metric-card-icon">
                      <TrendingUp size={18} />
                    </div>
                  </div>
                  <div className="metric-card-value">1,482</div>
                  <div className="metric-card-sub">Lượt tìm kiếm trong 30 ngày qua</div>
                </div>

                <div className="metric-card">
                  <div className="metric-card-bg-gradient" />
                  <div className="metric-card-header">
                    <span className="metric-card-title">Tìm kiếm lỗi (0 kết quả)</span>
                    <div className="metric-card-icon" style={{ backgroundColor: 'rgba(239, 68, 68, 0.1)', color: 'var(--danger)' }}>
                      <AlertCircle size={18} />
                    </div>
                  </div>
                  <div className="metric-card-value">42</div>
                  <div className="metric-card-sub">Cần AI phân tích tối ưu hóa</div>
                </div>

                <div className="metric-card">
                  <div className="metric-card-bg-gradient" />
                  <div className="metric-card-header">
                    <span className="metric-card-title">Tỷ lệ CTR Tìm kiếm</span>
                    <div className="metric-card-icon" style={{ backgroundColor: 'rgba(16, 185, 129, 0.1)', color: 'var(--success)' }}>
                      <CheckCircle2 size={18} />
                    </div>
                  </div>
                  <div className="metric-card-value">76.4%</div>
                  <div className="metric-card-sub">+3.2% so với tháng trước</div>
                </div>

                <div className="metric-card">
                  <div className="metric-card-bg-gradient" />
                  <div className="metric-card-header">
                    <span className="metric-card-title">Cấu hình Từ khóa</span>
                    <div className="metric-card-icon" style={{ backgroundColor: 'rgba(37, 99, 235, 0.1)', color: 'var(--primary)' }}>
                      <BookOpen size={18} />
                    </div>
                  </div>
                  <div className="metric-card-value">{spellcheckRules.length + synonymRules.length}</div>
                  <div className="metric-card-sub">{spellcheckRules.length} sửa lỗi, {synonymRules.length} đồng nghĩa</div>
                </div>
              </div>

              {/* Layout for analytics tables */}
              <div className="admin-grid" style={{ gridTemplateColumns: '1fr 1fr' }}>
                <section className="admin-card">
                  <div className="admin-card-title">
                    <AlertCircle size={18} style={{ color: 'var(--danger)' }} />
                    <span>Top Từ khóa 0 kết quả (Zero Results)</span>
                  </div>
                  <div className="table-wrapper">
                    <table className="admin-table">
                      <thead>
                        <tr>
                          <th>Từ khóa</th>
                          <th>Số lần tìm</th>
                          <th>Trạng thái đề xuất</th>
                        </tr>
                      </thead>
                      <tbody>
                        <tr>
                          <td style={{ fontWeight: 600 }}>stel</td>
                          <td>14</td>
                          <td><span className="badge-status" style={{ backgroundColor: 'var(--success-light)', color: 'var(--success)' }}>Đã gợi ý sửa đổi</span></td>
                        </tr>
                        <tr>
                          <td style={{ fontWeight: 600 }}>chuot gaming</td>
                          <td>12</td>
                          <td><span className="badge-status" style={{ backgroundColor: 'var(--success-light)', color: 'var(--success)' }}>Đã gợi ý sửa đổi</span></td>
                        </tr>
                        <tr>
                          <td style={{ fontWeight: 600 }}>keycron</td>
                          <td>9</td>
                          <td><span className="badge-status" style={{ backgroundColor: '#fef3c7', color: '#92400e' }}>Chờ AI quét</span></td>
                        </tr>
                      </tbody>
                    </table>
                  </div>
                </section>

                <section className="admin-card">
                  <div className="admin-card-title">
                    <TrendingUp size={18} style={{ color: 'var(--primary)' }} />
                    <span>Lượt tìm kiếm phổ biến theo danh mục</span>
                  </div>
                  <div className="table-wrapper">
                    <table className="admin-table">
                      <thead>
                        <tr>
                          <th>Danh mục</th>
                          <th>Số lượt tìm</th>
                          <th>Tỷ lệ Click (CTR)</th>
                        </tr>
                      </thead>
                      <tbody>
                        <tr>
                          <td style={{ fontWeight: 600 }}>Bàn phím cơ</td>
                          <td>642</td>
                          <td>82.1%</td>
                        </tr>
                        <tr>
                          <td style={{ fontWeight: 600 }}>Chuột Gaming</td>
                          <td>412</td>
                          <td>74.6%</td>
                        </tr>
                        <tr>
                          <td style={{ fontWeight: 600 }}>Tai nghe</td>
                          <td>198</td>
                          <td>68.9%</td>
                        </tr>
                      </tbody>
                    </table>
                  </div>
                </section>
              </div>

              {/* US-009 information alert */}
              <div 
                style={{ 
                  marginTop: '24px', 
                  padding: '16px', 
                  borderRadius: '8px', 
                  border: '1px solid var(--border-color)', 
                  backgroundColor: 'var(--bg-primary)', 
                  display: 'flex', 
                  gap: '12px', 
                  alignItems: 'center' 
                }}
              >
                <HelpCircle size={24} style={{ color: 'var(--primary)', flexShrink: 0 }} />
                <div>
                  <h4 style={{ fontWeight: 700, fontSize: '0.9rem', color: 'var(--text-primary)', marginBottom: '2px' }}>
                    Dashboard Analytics (US-009)
                  </h4>
                  <p style={{ fontSize: '0.8rem', color: 'var(--text-secondary)' }}>
                    Màn hình này đang hiển thị số liệu phân tích mẫu. Tính năng liên kết trực tiếp biểu đồ thời gian thực với dữ liệu từ ElasticSearch/OpenSearch clicklogs sẽ được hoàn thiện trong US-009.
                  </p>
                </div>
              </div>
            </div>
          )}

          {/* TAB 2: AI SUGGESTIONS ENGINE (BACKEND PAGINATED, FILTERED, SEARCHED) */}
          {activeTab === 'ai-suggestions' && (
            <div>
              {/* Header card with action triggers */}
              <div className="admin-card" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: '16px', marginBottom: '24px' }}>
                <div>
                  <h3 style={{ fontWeight: 700, fontSize: '1.1rem', marginBottom: '4px' }}>
                    Trung tâm đề xuất tối ưu hóa AI
                  </h3>
                  <p style={{ color: 'var(--text-secondary)', fontSize: '0.85rem' }}>
                    AI phân tích lịch sử tìm kiếm kém hiệu quả để đề xuất chính tả (Typo) hoặc từ đồng nghĩa (Synonym).
                  </p>
                </div>

                <div style={{ display: 'flex', gap: '12px', alignItems: 'center' }}>
                  <button 
                    onClick={handleGenerateAISuggestions}
                    disabled={generatingSuggestions || loadingTypos || loadingSynonyms}
                    className="btn btn-primary"
                    style={{ height: '36px', display: 'flex', alignItems: 'center', gap: '6px' }}
                  >
                    <Sparkles className={generatingSuggestions ? 'spinner' : ''} size={14} />
                    {generatingSuggestions ? 'Đang phân tích...' : 'Phân tích AI ngay'}
                  </button>

                  <button 
                    onClick={() => { fetchTypoSuggestions(); fetchSynonymSuggestions(); }}
                    disabled={loadingTypos || loadingSynonyms || generatingSuggestions}
                    className="btn btn-outline"
                    style={{ height: '36px', display: 'flex', alignItems: 'center', gap: '6px' }}
                  >
                    <RefreshCw className={(loadingTypos || loadingSynonyms) ? 'spinner' : ''} size={14} />
                    Tải lại
                  </button>
                </div>
              </div>

              {/* TABLE 1: TYPO CORRECTIONS */}
              <section className="admin-card" style={{ marginBottom: '24px' }}>
                <div className="table-controls">
                  <div className="admin-card-title" style={{ margin: 0 }}>
                    <AlertTriangle size={18} style={{ color: 'var(--warning)' }} />
                    <span>Đề xuất Sửa lỗi chính tả (Typo Corrections)</span>
                    <span className="badge-status" style={{ backgroundColor: 'var(--primary-light)', color: 'var(--primary)', marginLeft: '8px' }}>
                      {typoTotal} đề xuất
                    </span>
                  </div>

                  {/* Independent filter and search controls for Typo */}
                  <div style={{ display: 'flex', gap: '12px', alignItems: 'center', flexWrap: 'wrap' }}>
                    <select
                      value={typoStatus}
                      onChange={(e) => { setTypoStatus(e.target.value as any); setTypoPage(1); }}
                      className="btn btn-outline"
                      style={{ height: '38px', fontSize: '0.8rem', padding: '0 12px', borderRadius: 'var(--radius-sm)' }}
                    >
                      <option value="pending">Pending</option>
                      <option value="approved">Approved</option>
                      <option value="rejected">Rejected</option>
                    </select>

                    <div className="search-input-wrapper" style={{ margin: 0 }}>
                      <Search className="search-icon-inside" size={16} />
                      <input 
                        type="text"
                        placeholder="Tìm từ gốc hoặc gợi ý..."
                        value={typoSearch}
                        onChange={(e) => { setTypoSearch(e.target.value); setTypoPage(1); }}
                      />
                    </div>
                  </div>
                </div>

                <div className="table-wrapper" style={{ borderBottomLeftRadius: 0, borderBottomRightRadius: 0 }}>
                  <table className="admin-table">
                    <thead>
                      <tr>
                        <th>Từ gõ sai (Source)</th>
                        <th>Gợi ý thay thế (Suggested)</th>
                        <th>Độ tin cậy</th>
                        <th>Status</th>
                        <th>Hành động</th>
                      </tr>
                    </thead>
                    <tbody>
                      {typoSuggestions.length > 0 ? (
                        typoSuggestions.map((sugg) => (
                          <tr key={sugg.id}>
                            <td style={{ fontWeight: 600, color: 'var(--text-primary)' }}>{sugg.source_value}</td>
                            <td style={{ fontWeight: 700, color: 'var(--primary)' }}>{sugg.suggested_value}</td>
                            <td style={{ fontWeight: 600 }}>{(sugg.confidence_score * 100).toFixed(0)}%</td>
                            <td>
                              <span className={`badge-status ${sugg.status}`} style={{
                                backgroundColor: sugg.status === 'approved' ? 'var(--success-light)' : sugg.status === 'rejected' ? '#fee2e2' : 'var(--primary-light)',
                                color: sugg.status === 'approved' ? 'var(--success)' : sugg.status === 'rejected' ? 'var(--danger)' : 'var(--primary)'
                              }}>
                                {sugg.status.toUpperCase()}
                              </span>
                            </td>
                            <td>
                              {sugg.status === 'pending' && (
                                <div style={{ display: 'flex', gap: '8px' }}>
                                  <button 
                                    onClick={() => handleApproveSuggestion(sugg.id, 'typo')}
                                    className="btn btn-primary" 
                                    style={{ padding: '4px 10px', fontSize: '0.75rem', height: '28px', backgroundColor: 'var(--success)', borderColor: 'var(--success)' }}
                                  >
                                    Duyệt
                                  </button>
                                  <button 
                                    onClick={() => handleRejectSuggestion(sugg.id, 'typo')}
                                    className="btn btn-outline" 
                                    style={{ padding: '4px 10px', fontSize: '0.75rem', height: '28px', color: 'var(--danger)', borderColor: 'var(--danger)' }}
                                  >
                                    Từ chối
                                  </button>
                                </div>
                              )}
                            </td>
                          </tr>
                        ))
                      ) : (
                        <tr>
                          <td colSpan={5} style={{ textAlign: 'center', color: 'var(--text-muted)', fontStyle: 'italic', padding: '24px' }}>
                            {loadingTypos ? 'Đang tải đề xuất sửa lỗi...' : 'Không tìm thấy đề xuất chính tả nào.'}
                          </td>
                        </tr>
                      )}
                    </tbody>
                  </table>
                </div>

                {/* Typo Pagination */}
                <div className="pagination-container">
                  <div className="pagination-info">
                    Hiển thị {typoSuggestions.length > 0 ? typoStartIndex + 1 : 0} - {Math.min(typoStartIndex + itemsPerPage, typoTotal)} trong tổng số {typoTotal} đề xuất
                  </div>
                  <div className="pagination-buttons">
                    <button 
                      disabled={typoPage === 1}
                      onClick={() => setTypoPage(typoPage - 1)}
                      className="pagination-btn"
                    >
                      <ChevronLeft size={14} /> Trước
                    </button>
                    <span style={{ fontSize: '0.8rem', alignSelf: 'center', fontWeight: 600, color: 'var(--text-secondary)', padding: '0 8px' }}>
                      Trang {typoPage} / {totalTypoPages}
                    </span>
                    <button 
                      disabled={typoPage === totalTypoPages}
                      onClick={() => setTypoPage(typoPage + 1)}
                      className="pagination-btn"
                    >
                      Sau <ChevronRight size={14} />
                    </button>
                  </div>
                </div>
              </section>

              {/* TABLE 2: SYNONYM SUGGESTIONS */}
              <section className="admin-card">
                <div className="table-controls">
                  <div className="admin-card-title" style={{ margin: 0 }}>
                    <Sparkles size={18} style={{ color: 'var(--primary)' }} />
                    <span>Đề xuất Từ đồng nghĩa (Synonyms Proposals)</span>
                    <span className="badge-status" style={{ backgroundColor: 'rgba(59, 130, 246, 0.1)', color: 'var(--secondary)', marginLeft: '8px' }}>
                      {synonymTotal} đề xuất
                    </span>
                  </div>

                  {/* Independent filter and search controls for Synonym */}
                  <div style={{ display: 'flex', gap: '12px', alignItems: 'center', flexWrap: 'wrap' }}>
                    <select
                      value={synonymStatus}
                      onChange={(e) => { setSynonymStatus(e.target.value as any); setSynonymPage(1); }}
                      className="btn btn-outline"
                      style={{ height: '38px', fontSize: '0.8rem', padding: '0 12px', borderRadius: 'var(--radius-sm)' }}
                    >
                      <option value="pending">Pending</option>
                      <option value="approved">Approved</option>
                      <option value="rejected">Rejected</option>
                    </select>

                    <div className="search-input-wrapper" style={{ margin: 0 }}>
                      <Search className="search-icon-inside" size={16} />
                      <input 
                        type="text"
                        placeholder="Tìm từ gốc hoặc gợi ý..."
                        value={synonymSearch}
                        onChange={(e) => { setSynonymSearch(e.target.value); setSynonymPage(1); }}
                      />
                    </div>
                  </div>
                </div>

                <div className="table-wrapper" style={{ borderBottomLeftRadius: 0, borderBottomRightRadius: 0 }}>
                  <table className="admin-table">
                    <thead>
                      <tr>
                        <th>Từ gốc (Source)</th>
                        <th>Gợi ý đồng nghĩa (Synonym)</th>
                        <th>Độ tin cậy</th>
                        <th>Status</th>
                        <th>Hành động</th>
                      </tr>
                    </thead>
                    <tbody>
                      {synonymSuggestions.length > 0 ? (
                        synonymSuggestions.map((sugg) => (
                          <tr key={sugg.id}>
                            <td style={{ fontWeight: 600, color: 'var(--text-primary)' }}>{sugg.source_value}</td>
                            <td style={{ fontWeight: 700, color: 'var(--success)' }}>{sugg.suggested_value}</td>
                            <td style={{ fontWeight: 600 }}>{(sugg.confidence_score * 100).toFixed(0)}%</td>
                            <td>
                              <span className={`badge-status ${sugg.status}`} style={{
                                backgroundColor: sugg.status === 'approved' ? 'var(--success-light)' : sugg.status === 'rejected' ? '#fee2e2' : 'var(--primary-light)',
                                color: sugg.status === 'approved' ? 'var(--success)' : sugg.status === 'rejected' ? 'var(--danger)' : 'var(--primary)'
                              }}>
                                {sugg.status.toUpperCase()}
                              </span>
                            </td>
                            <td>
                              {sugg.status === 'pending' && (
                                <div style={{ display: 'flex', gap: '8px' }}>
                                  <button 
                                    onClick={() => handleApproveSuggestion(sugg.id, 'synonym')}
                                    className="btn btn-primary" 
                                    style={{ padding: '4px 10px', fontSize: '0.75rem', height: '28px', backgroundColor: 'var(--success)', borderColor: 'var(--success)' }}
                                  >
                                    Duyệt
                                  </button>
                                  <button 
                                    onClick={() => handleRejectSuggestion(sugg.id, 'synonym')}
                                    className="btn btn-outline" 
                                    style={{ padding: '4px 10px', fontSize: '0.75rem', height: '28px', color: 'var(--danger)', borderColor: 'var(--danger)' }}
                                  >
                                    Từ chối
                                  </button>
                                </div>
                              )}
                            </td>
                          </tr>
                        ))
                      ) : (
                        <tr>
                          <td colSpan={5} style={{ textAlign: 'center', color: 'var(--text-muted)', fontStyle: 'italic', padding: '24px' }}>
                            {loadingSynonyms ? 'Đang tải đề xuất đồng nghĩa...' : 'Không tìm thấy đề xuất đồng nghĩa nào.'}
                          </td>
                        </tr>
                      )}
                    </tbody>
                  </table>
                </div>

                {/* Synonym Pagination */}
                <div className="pagination-container">
                  <div className="pagination-info">
                    Hiển thị {synonymSuggestions.length > 0 ? synonymStartIndex + 1 : 0} - {Math.min(synonymStartIndex + itemsPerPage, synonymTotal)} trong tổng số {synonymTotal} đề xuất
                  </div>
                  <div className="pagination-buttons">
                    <button 
                      disabled={synonymPage === 1}
                      onClick={() => setSynonymPage(synonymPage - 1)}
                      className="pagination-btn"
                    >
                      <ChevronLeft size={14} /> Trước
                    </button>
                    <span style={{ fontSize: '0.8rem', alignSelf: 'center', fontWeight: 600, color: 'var(--text-secondary)', padding: '0 8px' }}>
                      Trang {synonymPage} / {totalSynonymPages}
                    </span>
                    <button 
                      disabled={synonymPage === totalSynonymPages}
                      onClick={() => setSynonymPage(synonymPage + 1)}
                      className="pagination-btn"
                    >
                      Sau <ChevronRight size={14} />
                    </button>
                  </div>
                </div>
              </section>
            </div>
          )}

          {/* TAB 3: DICTIONARIES (ACTIVE SPELLCHECK & SYNONYMS) */}
          {activeTab === 'dictionaries' && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
              <section className="admin-card">
                <div className="admin-card-title" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', width: '100%', borderBottom: '1px solid var(--border-color)', paddingBottom: '16px', marginBottom: '16px' }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                    <BookOpen size={18} style={{ color: 'var(--primary)' }} />
                    <span>Danh sách cấu hình quy tắc từ khóa (Active Rules)</span>
                  </div>
                  <button 
                    onClick={() => setShowAddRuleModal(true)} 
                    className="btn btn-primary" 
                    style={{ display: 'flex', gap: '6px', alignItems: 'center', height: '36px', fontSize: '0.85rem' }}
                  >
                    <Plus size={16} /> Thêm quy tắc mới
                  </button>
                </div>
                <p style={{ fontSize: '0.85rem', color: 'var(--text-secondary)', marginBottom: '24px' }}>
                  Đây là các luật sửa chính tả và từ đồng nghĩa đang được hệ thống Search Engine áp dụng trực tiếp khi khách hàng gõ từ khóa.
                </p>

                {/* Synonyms list */}
                <div style={{ marginBottom: '32px' }}>
                  <h3 style={{ fontSize: '0.95rem', fontWeight: 700, marginBottom: '12px', display: 'flex', alignItems: 'center', gap: '6px', color: 'var(--text-primary)' }}>
                    <Sparkles size={16} style={{ color: 'var(--primary)' }} /> Quy tắc từ đồng nghĩa (Active Synonyms)
                  </h3>
                  <div className="table-wrapper">
                    <table className="admin-table">
                      <thead>
                        <tr>
                          <th>Từ khóa chính (Keyword)</th>
                          <th>Các từ đồng nghĩa tương đương (Synonyms)</th>
                          <th>Status</th>
                          <th style={{ width: '100px', textAlign: 'center' }}>Hành động</th>
                        </tr>
                      </thead>
                      <tbody>
                        {synonymRules.length > 0 ? (
                          synonymRules.map((rule) => (
                            <tr key={rule.id}>
                              <td style={{ fontWeight: 600, color: 'var(--text-primary)' }}>{rule.keyword}</td>
                              <td style={{ fontWeight: 600, color: 'var(--success)' }}>{rule.synonym}</td>
                              <td>
                                <span className="badge-status active">ACTIVE</span>
                              </td>
                              <td style={{ textAlign: 'center' }}>
                                <button 
                                  onClick={() => handleDeleteSynonym(rule.id)}
                                  className="btn btn-outline" 
                                  style={{ padding: '4px 8px', color: 'var(--danger)', borderColor: 'rgba(239, 68, 68, 0.2)', height: '28px', fontSize: '0.75rem', fontWeight: 600 }}
                                >
                                  Xóa
                                </button>
                              </td>
                            </tr>
                          ))
                        ) : (
                          <tr>
                            <td colSpan={4} style={{ textAlign: 'center', color: 'var(--text-muted)', fontStyle: 'italic', padding: '16px' }}>
                              {loadingDictionaries ? 'Đang tải quy tắc đồng nghĩa...' : 'Chưa có quy tắc từ đồng nghĩa nào hoạt động.'}
                            </td>
                          </tr>
                        )}
                      </tbody>
                    </table>
                  </div>
                </div>

                {/* Spellcheck list */}
                <div>
                  <h3 style={{ fontSize: '0.95rem', fontWeight: 700, marginBottom: '12px', display: 'flex', alignItems: 'center', gap: '6px', color: 'var(--text-primary)' }}>
                    <AlertTriangle size={16} style={{ color: 'var(--warning)' }} /> Quy tắc sửa lỗi chính tả (Active Spellcheck)
                  </h3>
                  <div className="table-wrapper">
                    <table className="admin-table">
                      <thead>
                        <tr>
                          <th>Từ gõ sai (Typo Word)</th>
                          <th>Từ sửa đúng (Correct Word)</th>
                          <th>Status</th>
                          <th style={{ width: '100px', textAlign: 'center' }}>Hành động</th>
                        </tr>
                      </thead>
                      <tbody>
                        {spellcheckRules.length > 0 ? (
                          spellcheckRules.map((rule) => (
                            <tr key={rule.id}>
                              <td style={{ fontWeight: 600, color: 'var(--text-primary)' }}>{rule.typo_word}</td>
                              <td style={{ fontWeight: 600, color: 'var(--primary)' }}>{rule.correct_word}</td>
                              <td>
                                <span className="badge-status active">ACTIVE</span>
                              </td>
                              <td style={{ textAlign: 'center' }}>
                                <button 
                                  onClick={() => handleDeleteSpellcheck(rule.id)}
                                  className="btn btn-outline" 
                                  style={{ padding: '4px 8px', color: 'var(--danger)', borderColor: 'rgba(239, 68, 68, 0.2)', height: '28px', fontSize: '0.75rem', fontWeight: 600 }}
                                >
                                  Xóa
                                </button>
                              </td>
                            </tr>
                          ))
                        ) : (
                          <tr>
                            <td colSpan={4} style={{ textAlign: 'center', color: 'var(--text-muted)', fontStyle: 'italic', padding: '16px' }}>
                              {loadingDictionaries ? 'Đang tải từ điển sửa lỗi...' : 'Chưa có quy tắc sửa chính tả nào hoạt động.'}
                            </td>
                          </tr>
                        )}
                      </tbody>
                    </table>
                  </div>
                </div>
              </section>
            </div>
          )}

          {/* TAB 4: DATABASE SYNC & LOGS */}
          {activeTab === 'sync' && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
              <div className="admin-grid" style={{ gridTemplateColumns: '320px 1fr' }}>
                {/* Sync controls */}
                <section className="admin-card">
                  <div className="admin-card-title">
                    <RefreshCw size={18} style={{ color: 'var(--primary)' }} />
                    <span>Kích hoạt đồng bộ</span>
                  </div>
                  <p style={{ fontSize: '0.8rem', color: 'var(--text-secondary)', marginBottom: '20px' }}>
                    Đồng bộ hóa toàn bộ sản phẩm của Tenant từ Postgres sang OpenSearch index.
                  </p>

                  <button
                    disabled={syncing}
                    onClick={handleSyncDatabase}
                    className="btn btn-primary"
                    style={{ width: '100%', height: '46px', display: 'flex', alignItems: 'center', gap: '8px', justifyContent: 'center' }}
                  >
                    {syncing ? (
                      <>
                        <RefreshCw className="spinner" size={18} />
                        Đang đồng bộ...
                      </>
                    ) : (
                      <>
                        <RefreshCw size={18} />
                        Đồng bộ ngay
                      </>
                    )}
                  </button>

                  {syncMessage && (
                    <div 
                      style={{ 
                        marginTop: '16px', 
                        padding: '12px', 
                        borderRadius: '8px', 
                        fontSize: '0.8rem', 
                        display: 'flex', 
                        gap: '8px',
                        backgroundColor: syncMessage.type === 'success' ? 'var(--success-light)' : '#fee2e2',
                        color: syncMessage.type === 'success' ? 'var(--success)' : 'var(--danger)',
                        border: `1px solid ${syncMessage.type === 'success' ? 'rgba(16, 185, 129, 0.2)' : 'rgba(239, 68, 68, 0.2)'}`
                      }}
                    >
                      {syncMessage.type === 'success' ? (
                        <CheckCircle2 size={16} style={{ flexShrink: 0 }} />
                      ) : (
                        <AlertTriangle size={16} style={{ flexShrink: 0 }} />
                      )}
                      <span>{syncMessage.text}</span>
                    </div>
                  )}
                </section>

                {/* Sync Logs list */}
                <section className="admin-card">
                  <div className="admin-card-title" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                      <Clock size={18} style={{ color: 'var(--primary)' }} />
                      <span>Nhật ký lịch sử đồng bộ (Sync Logs)</span>
                    </div>
                    {syncLogs.length > 0 && (
                      <button 
                        onClick={clearSyncLogs} 
                        style={{ fontSize: '0.75rem', color: 'var(--danger)', cursor: 'pointer', fontWeight: 600, background: 'none', border: 'none' }}
                      >
                        Xóa lịch sử
                      </button>
                    )}
                  </div>
                  <p style={{ fontSize: '0.8rem', color: 'var(--text-secondary)', marginBottom: '16px' }}>
                    Lịch sử chạy đồng bộ Postgres sang OpenSearch thủ công của Admin.
                  </p>

                  <div className="table-wrapper">
                    <table className="admin-table">
                      <thead>
                        <tr>
                          <th>Thời gian chạy</th>
                          <th>Cửa hàng (Tenant)</th>
                          <th>Số lượng cập nhật</th>
                          <th>Kết quả trạng thái</th>
                        </tr>
                      </thead>
                      <tbody>
                        {syncLogs.length > 0 ? (
                          syncLogs.map((log, idx) => (
                            <tr key={idx}>
                              <td>{log.timestamp}</td>
                              <td>{log.tenantName}</td>
                              <td style={{ fontWeight: 600 }}>{log.syncedCount} SP</td>
                              <td>
                                <span 
                                  className="badge-status" 
                                  style={{ 
                                    backgroundColor: log.status === 'SUCCESS' ? 'var(--success-light)' : '#fee2e2',
                                    color: log.status === 'SUCCESS' ? 'var(--success)' : 'var(--danger)'
                                  }}
                                >
                                  {log.status === 'SUCCESS' ? 'THÀNH CÔNG' : 'THẤT BẠI'}
                                </span>
                              </td>
                            </tr>
                          ))
                        ) : (
                          <tr>
                            <td colSpan={4} style={{ textAlign: 'center', color: 'var(--text-muted)', fontStyle: 'italic', padding: '24px' }}>
                              Chưa ghi nhận lịch sử đồng bộ dữ liệu nào.
                            </td>
                          </tr>
                        )}
                      </tbody>
                    </table>
                  </div>
                </section>
              </div>
            </div>
          )}
        </main>

        <Footer />
      </div>

      {showAddRuleModal && (
        <div style={{
          position: 'fixed',
          top: 0,
          left: 0,
          right: 0,
          bottom: 0,
          backgroundColor: 'rgba(0, 0, 0, 0.5)',
          display: 'flex',
          justifyContent: 'center',
          alignItems: 'center',
          zIndex: 1000,
          backdropFilter: 'blur(4px)'
        }}>
          <div className="admin-card" style={{
            width: '100%',
            maxWidth: '500px',
            margin: '20px',
            position: 'relative',
            boxShadow: 'var(--shadow-lg)'
          }}>
            <button 
              onClick={() => setShowAddRuleModal(false)}
              style={{
                position: 'absolute',
                top: '16px',
                right: '16px',
                background: 'none',
                border: 'none',
                cursor: 'pointer',
                color: 'var(--text-secondary)'
              }}
            >
              <X size={20} />
            </button>

            <div className="admin-card-title" style={{ marginBottom: '20px' }}>
              <Plus size={18} style={{ color: 'var(--primary)' }} />
              <span>Thêm Quy tắc Từ điển Thủ công</span>
            </div>

            <form onSubmit={handleAddRule}>
              <div style={{ marginBottom: '16px' }}>
                <label style={{ display: 'block', fontSize: '0.85rem', fontWeight: 600, marginBottom: '6px', color: 'var(--text-secondary)' }}>
                  Loại quy tắc
                </label>
                <div style={{ display: 'flex', gap: '16px' }}>
                  <label style={{ display: 'flex', gap: '8px', alignItems: 'center', fontSize: '0.85rem', cursor: 'pointer' }}>
                    <input 
                      type="radio" 
                      name="rule_type" 
                      checked={newRuleType === 'synonym'} 
                      onChange={() => setNewRuleType('synonym')} 
                    />
                    Từ đồng nghĩa (Synonym)
                  </label>
                  <label style={{ display: 'flex', gap: '8px', alignItems: 'center', fontSize: '0.85rem', cursor: 'pointer' }}>
                    <input 
                      type="radio" 
                      name="rule_type" 
                      checked={newRuleType === 'spellcheck'} 
                      onChange={() => setNewRuleType('spellcheck')} 
                    />
                    Sửa lỗi chính tả (Spellcheck)
                  </label>
                </div>
              </div>

              {newRuleType === 'synonym' ? (
                <>
                  <div style={{ marginBottom: '16px' }}>
                    <label style={{ display: 'block', fontSize: '0.85rem', fontWeight: 600, marginBottom: '6px', color: 'var(--text-secondary)' }}>
                      Từ khóa chính (Keyword)
                    </label>
                    <input 
                      type="text" 
                      value={newKeyword}
                      onChange={(e) => setNewKeyword(e.target.value)}
                      placeholder="Ví dụ: keyboard"
                      style={{
                        width: '100%',
                        height: '40px',
                        padding: '0 12px',
                        borderRadius: 'var(--radius-sm)',
                        border: '1.5px solid var(--border-color)',
                        fontSize: '0.85rem',
                        backgroundColor: 'var(--bg-primary)',
                        color: 'var(--text-primary)'
                      }}
                      required
                    />
                  </div>
                  <div style={{ marginBottom: '16px' }}>
                    <label style={{ display: 'block', fontSize: '0.85rem', fontWeight: 600, marginBottom: '6px', color: 'var(--text-secondary)' }}>
                      Từ đồng nghĩa (Synonym)
                    </label>
                    <input 
                      type="text" 
                      value={newSynonym}
                      onChange={(e) => setNewSynonym(e.target.value)}
                      placeholder="Ví dụ: bàn phím"
                      style={{
                        width: '100%',
                        height: '40px',
                        padding: '0 12px',
                        borderRadius: 'var(--radius-sm)',
                        border: '1.5px solid var(--border-color)',
                        fontSize: '0.85rem',
                        backgroundColor: 'var(--bg-primary)',
                        color: 'var(--text-primary)'
                      }}
                      required
                    />
                  </div>
                  <div style={{ marginBottom: '20px' }}>
                    <label style={{ display: 'flex', gap: '8px', alignItems: 'center', fontSize: '0.85rem', cursor: 'pointer', fontWeight: 500, color: 'var(--text-secondary)' }}>
                      <input 
                        type="checkbox" 
                        checked={isBidirectional}
                        onChange={(e) => setIsBidirectional(e.target.checked)}
                      />
                      Áp dụng quan hệ hai chiều (Bidirectional)
                    </label>
                    <span style={{ display: 'block', fontSize: '0.75rem', color: 'var(--text-muted)', marginTop: '4px', marginLeft: '22px' }}>
                      (Hệ thống tự động sinh quy tắc ngược lại: A &rarr; B và B &rarr; A)
                    </span>
                  </div>
                </>
              ) : (
                <>
                  <div style={{ marginBottom: '16px' }}>
                    <label style={{ display: 'block', fontSize: '0.85rem', fontWeight: 600, marginBottom: '6px', color: 'var(--text-secondary)' }}>
                      Từ gõ sai (Typo Word)
                    </label>
                    <input 
                      type="text" 
                      value={newTypoWord}
                      onChange={(e) => setNewTypoWord(e.target.value)}
                      placeholder="Ví dụ: ban phin"
                      style={{
                        width: '100%',
                        height: '40px',
                        padding: '0 12px',
                        borderRadius: 'var(--radius-sm)',
                        border: '1.5px solid var(--border-color)',
                        fontSize: '0.85rem',
                        backgroundColor: 'var(--bg-primary)',
                        color: 'var(--text-primary)'
                      }}
                      required
                    />
                  </div>
                  <div style={{ marginBottom: '20px' }}>
                    <label style={{ display: 'block', fontSize: '0.85rem', fontWeight: 600, marginBottom: '6px', color: 'var(--text-secondary)' }}>
                      Từ sửa đúng (Correct Word)
                    </label>
                    <input 
                      type="text" 
                      value={newCorrectWord}
                      onChange={(e) => setNewCorrectWord(e.target.value)}
                      placeholder="Ví dụ: bàn phím"
                      style={{
                        width: '100%',
                        height: '40px',
                        padding: '0 12px',
                        borderRadius: 'var(--radius-sm)',
                        border: '1.5px solid var(--border-color)',
                        fontSize: '0.85rem',
                        backgroundColor: 'var(--bg-primary)',
                        color: 'var(--text-primary)'
                      }}
                      required
                    />
                  </div>
                </>
              )}

              <div style={{ display: 'flex', gap: '12px', justifyContent: 'flex-end' }}>
                <button 
                  type="button" 
                  onClick={() => setShowAddRuleModal(false)} 
                  className="btn btn-outline"
                  style={{ height: '38px', fontSize: '0.85rem' }}
                >
                  Hủy
                </button>
                <button 
                  type="submit" 
                  disabled={actionLoading}
                  className="btn btn-primary"
                  style={{ height: '38px', fontSize: '0.85rem', minWidth: '100px' }}
                >
                  {actionLoading ? 'Đang tạo...' : 'Lưu quy tắc'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};

export default Admin;
