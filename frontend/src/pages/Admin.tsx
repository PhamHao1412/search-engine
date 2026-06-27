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
  AlertTriangle 
} from 'lucide-react';
import { useTenant } from '../context/TenantContext';
import { searchApi } from '../services/api';
import Footer from '../components/Footer';

interface SyncLog {
  timestamp: string;
  tenantId: string;
  tenantName: string;
  syncedCount: number;
  status: 'SUCCESS' | 'FAILED';
  errorMessage?: string;
}

// Static dictionary rules display based on active tenant ID
const SYNONYMS_RULES: Record<string, Array<{ keyword: string; synonyms: string; status: string }>> = {
  'd3b07384-d113-4956-a5db-251d50c18d01': [
    { keyword: 'akko', synonyms: 'bàn phím, keyboard, phím cơ, keycap', status: 'active' },
    { keyword: 'logitech', synonyms: 'chuột, mouse, gaming gear, bàn di chuột', status: 'active' },
    { keyword: 'bàn phím', synonyms: 'keyboard, phím cơ, keycap, gaming keyboard', status: 'active' },
    { keyword: 'chuột', synonyms: 'mouse, chuột không dây, gaming mouse', status: 'active' },
  ],
  'a1a2a3a4-b1b2-c1c2-d1d2-e1e2e3e4e5e6': [
    { keyword: 'son', synonyms: 'lipstick, son dưỡng, mỹ phẩm, lip balm', status: 'active' },
    { keyword: 'mỹ phẩm', synonyms: 'cosmetics, son môi, kem dưỡng da, skincare', status: 'active' },
    { keyword: 'kem chống nắng', synonyms: 'sunscreen, bảo vệ da, mỹ phẩm', status: 'active' },
  ]
};

const SPELLCHECK_RULES: Record<string, Array<{ typo: string; correct: string; status: string }>> = {
  'd3b07384-d113-4956-a5db-251d50c18d01': [
    { typo: 'ako', correct: 'akko', status: 'active' },
    { typo: 'logitek', correct: 'logitech', status: 'active' },
    { typo: 'chuot', correct: 'chuột', status: 'active' },
    { typo: 'ban phim', correct: 'bàn phím', status: 'active' },
  ],
  'a1a2a3a4-b1b2-c1c2-d1d2-e1e2e3e4e5e6': [
    { typo: 'son duong', correct: 'son dưỡng', status: 'active' },
    { typo: 'my pham', correct: 'mỹ phẩm', status: 'active' },
    { typo: 'lip balm', correct: 'son dưỡng', status: 'active' },
  ]
};

const Admin: React.FC = () => {
  const navigate = useNavigate();
  const { activeTenant, setActiveTenantById, tenants } = useTenant();

  // Sync state
  const [syncing, setSyncing] = useState(false);
  const [syncMessage, setSyncMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);
  const [syncLogs, setSyncLogs] = useState<SyncLog[]>([]);

  // Load sync logs from localStorage
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

      const updatedLogs = [newLog, ...syncLogs].slice(0, 10); // Keep last 10 logs
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

  const synonyms = SYNONYMS_RULES[activeTenant.id] || [];
  const spellcheck = SPELLCHECK_RULES[activeTenant.id] || [];

  return (
    <div className="app-container">
      {/* HEADER SECTION */}
      <header className="header">
        <div className="header-logo">
          <Settings className="text-gradient" size={26} strokeWidth={2.5} />
          <span>Amaze<span style={{ color: 'var(--primary)' }}>Search</span> <span style={{ fontSize: '0.85rem', fontWeight: 500, color: 'var(--text-muted)' }}>Admin Panel</span></span>
        </div>

        <div className="header-actions">
          <button onClick={() => navigate('/')} className="btn btn-outline">
            <ArrowLeft size={16} />
            Quay Lại Cửa Hàng
          </button>
        </div>
      </header>

      <main className="main-content">
        <div style={{ marginBottom: '24px' }}>
          <h1 style={{ fontWeight: 800, fontSize: '1.8rem', letterSpacing: '-0.02em', marginBottom: '4px' }}>
            Hệ thống Quản trị Search Engine
          </h1>
          <p style={{ color: 'var(--text-secondary)', fontSize: '0.9rem' }}>
            Cấu hình tenant hoạt động, đồng bộ sản phẩm từ cơ sở dữ liệu lên OpenSearch và theo dõi từ điển tìm kiếm.
          </p>
        </div>

        {/* ADMIN GRID LAYOUT */}
        <div className="admin-grid">
          
          {/* LEFT SIDEBAR: TENANT SWITCHER & SYNC PANEL */}
          <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
            
            {/* Tenant switcher manager */}
            <section className="admin-card">
              <div className="admin-card-title">
                <Server size={18} style={{ color: 'var(--primary)' }} />
                <span>Quản lý Tenant Active</span>
              </div>
              <p style={{ fontSize: '0.8rem', color: 'var(--text-secondary)', marginBottom: '16px' }}>
                Thay đổi Tenant sẽ lập tức chuyển đổi phạm vi index tìm kiếm trên trang mua sắm.
              </p>

              <div className="tenant-list">
                {tenants.map((t) => (
                  <div
                    key={t.id}
                    onClick={() => setActiveTenantById(t.id)}
                    className={`tenant-option ${activeTenant.id === t.id ? 'active' : ''}`}
                  >
                    <div className="tenant-indicator">
                      <div className="tenant-indicator-dot" />
                    </div>
                    <div className="tenant-info">
                      <span className="tenant-name">{t.name}</span>
                      <span className="tenant-id">{t.id.substring(0, 8)}...</span>
                    </div>
                  </div>
                ))}
              </div>
            </section>

            {/* Sync control database panel */}
            <section className="admin-card">
              <div className="admin-card-title">
                <RefreshCw size={18} style={{ color: 'var(--primary)' }} />
                <span>Đồng bộ Cơ sở dữ liệu</span>
              </div>
              <p style={{ fontSize: '0.8rem', color: 'var(--text-secondary)', marginBottom: '16px' }}>
                Đẩy dữ liệu sản phẩm mới nhất từ PostgreSQL của Tenant qua Elasticsearch/OpenSearch.
              </p>

              <button
                disabled={syncing}
                onClick={handleSyncDatabase}
                className="btn btn-primary"
                style={{ width: '100%', height: '46px' }}
              >
                {syncing ? (
                  <>
                    <RefreshCw className="spinner" size={18} />
                    Đang đồng bộ...
                  </>
                ) : (
                  <>
                    <RefreshCw size={18} />
                    Sync Database
                  </>
                )}
              </button>

              {/* Status Message feedback */}
              {syncMessage && (
                <div 
                  style={{ 
                    marginTop: '16px', 
                    padding: '12px', 
                    borderRadius: '8px', 
                    fontSize: '0.8rem', 
                    display: 'flex', 
                    gap: '8px',
                    alignItems: 'flex-start',
                    backgroundColor: syncMessage.type === 'success' ? 'var(--success-light)' : '#fee2e2',
                    color: syncMessage.type === 'success' ? 'var(--success)' : 'var(--danger)',
                    border: `1px solid ${syncMessage.type === 'success' ? 'rgba(16, 185, 129, 0.2)' : 'rgba(239, 68, 68, 0.2)'}`
                  }}
                >
                  {syncMessage.type === 'success' ? (
                    <CheckCircle2 size={16} style={{ flexShrink: 0, marginTop: '2px' }} />
                  ) : (
                    <AlertTriangle size={16} style={{ flexShrink: 0, marginTop: '2px' }} />
                  )}
                  <span>{syncMessage.text}</span>
                </div>
              )}
            </section>
          </div>

          {/* RIGHT AREA: DICTIONARIES & LOGS TABLES */}
          <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
            
            {/* Search Dictionaries (Synonyms & Spellchecks) */}
            <section className="admin-card">
              <div className="admin-card-title">
                <BookOpen size={18} style={{ color: 'var(--primary)' }} />
                <span>Từ điển Tìm kiếm (Search Dictionaries)</span>
              </div>
              <p style={{ fontSize: '0.85rem', color: 'var(--text-secondary)', marginBottom: '16px' }}>
                Đang hiển thị từ điển cấu hình của <strong style={{ color: 'var(--primary)' }}>{activeTenant.name}</strong>.
              </p>

              {/* Synonyms Tab Table */}
              <div style={{ marginBottom: '24px' }}>
                <h3 style={{ fontSize: '0.9rem', fontWeight: 600, marginBottom: '8px', display: 'flex', alignItems: 'center', gap: '6px' }}>
                  <Sparkles size={14} style={{ color: 'var(--primary)' }} /> Từ đồng nghĩa (Synonyms Expansion)
                </h3>
                <div className="table-wrapper">
                  <table className="admin-table">
                    <thead>
                      <tr>
                        <th>Từ khóa chính</th>
                        <th>Từ đồng nghĩa tương đương</th>
                        <th>Trạng thái</th>
                      </tr>
                    </thead>
                    <tbody>
                      {synonyms.length > 0 ? (
                        synonyms.map((rule, idx) => (
                          <tr key={idx}>
                            <td style={{ fontWeight: 600, color: 'var(--text-primary)' }}>{rule.keyword}</td>
                            <td>{rule.synonyms}</td>
                            <td>
                              <span className="badge-status active">Hoạt động</span>
                            </td>
                          </tr>
                        ))
                      ) : (
                        <tr>
                          <td colSpan={3} style={{ textAlign: 'center', color: 'var(--text-muted)', fontStyle: 'italic' }}>
                            Không có quy tắc từ đồng nghĩa nào được cấu hình.
                          </td>
                        </tr>
                      )}
                    </tbody>
                  </table>
                </div>
              </div>

              {/* Spellcheck Tab Table */}
              <div>
                <h3 style={{ fontSize: '0.9rem', fontWeight: 600, marginBottom: '8px', display: 'flex', alignItems: 'center', gap: '6px' }}>
                  <Sparkles size={14} style={{ color: 'var(--primary)' }} /> Sửa lỗi chính tả (Spellcheck Dictionary)
                </h3>
                <div className="table-wrapper">
                  <table className="admin-table">
                    <thead>
                      <tr>
                        <th>Từ gõ sai (Typo Word)</th>
                        <th>Từ sửa đúng (Correct Word)</th>
                        <th>Trạng thái</th>
                      </tr>
                    </thead>
                    <tbody>
                      {spellcheck.length > 0 ? (
                        spellcheck.map((rule, idx) => (
                          <tr key={idx}>
                            <td style={{ fontWeight: 600, color: 'var(--text-primary)' }}>{rule.typo}</td>
                            <td>{rule.correct}</td>
                            <td>
                              <span className="badge-status active">Hoạt động</span>
                            </td>
                          </tr>
                        ))
                      ) : (
                        <tr>
                          <td colSpan={3} style={{ textAlign: 'center', color: 'var(--text-muted)', fontStyle: 'italic' }}>
                            Không có quy tắc chính tả nào được cấu hình.
                          </td>
                        </tr>
                      )}
                    </tbody>
                  </table>
                </div>
              </div>
            </section>

            {/* Sync History Logs List */}
            <section className="admin-card">
              <div className="admin-card-title" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                  <Clock size={18} style={{ color: 'var(--primary)' }} />
                  <span>Lịch sử Đồng bộ (Sync History Logs)</span>
                </div>
                {syncLogs.length > 0 && (
                  <button 
                    onClick={clearSyncLogs} 
                    style={{ fontSize: '0.75rem', color: 'var(--danger)', cursor: 'pointer', fontWeight: 500 }}
                  >
                    Xóa lịch sử
                  </button>
                )}
              </div>
              <p style={{ fontSize: '0.8rem', color: 'var(--text-secondary)', marginBottom: '16px' }}>
                Nhật ký các lần kích hoạt API đồng bộ database với OpenSearch.
              </p>

              <div className="table-wrapper">
                <table className="admin-table">
                  <thead>
                    <tr>
                      <th>Thời gian</th>
                      <th>Cửa hàng (Tenant)</th>
                      <th>Số lượng cập nhật</th>
                      <th>Kết quả</th>
                    </tr>
                  </thead>
                  <tbody>
                    {syncLogs.length > 0 ? (
                      syncLogs.map((log, idx) => (
                        <tr key={idx}>
                          <td>{log.timestamp}</td>
                          <td style={{ fontSize: '0.8rem' }}>{log.tenantName}</td>
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
                          Chưa có lịch sử đồng bộ nào được ghi nhận.
                        </td>
                      </tr>
                    )}
                  </tbody>
                </table>
              </div>
            </section>
          </div>

        </div>
      </main>
      <Footer />
    </div>
  );
};

export default Admin;
