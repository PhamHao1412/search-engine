import React, { createContext, useContext, useState, useEffect } from 'react';
import { adminApi } from '../services/api';

export interface Tenant {
  id: string;
  name: string;
  description?: string;
}

const DEFAULT_TENANTS: Tenant[] = [
  {
    id: 'd3b07384-d113-4956-a5db-251d50c18d01',
    name: 'Tenant A (PC & Bàn Phím)',
    description: 'Bàn phím cơ, chuột máy tính và thiết bị công nghệ.',
  },
  {
    id: '25f2c7e4-92d7-4efb-a710-7bf77bc479a2',
    name: 'Tenant B (Mỹ phẩm & Chăm sóc)',
    description: 'Mỹ phẩm, son môi và các thiết bị làm đẹp.',
  },
];

interface TenantContextType {
  activeTenant: Tenant;
  setActiveTenantById: (id: string) => void;
  tenants: Tenant[];
}

const TenantContext = createContext<TenantContextType | undefined>(undefined);

export const TenantProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [tenants, setTenants] = useState<Tenant[]>(DEFAULT_TENANTS);
  const [activeTenant, setActiveTenant] = useState<Tenant>(DEFAULT_TENANTS[0]);

  useEffect(() => {
    const loadTenantsFromDB = async () => {
      try {
        const dbTenants = await adminApi.getTenants();
        if (dbTenants && dbTenants.length > 0) {
          const normalized: Tenant[] = dbTenants.map((t) => ({
            id: t.id,
            name: t.name,
            description: t.id === 'd3b07384-d113-4956-a5db-251d50c18d01'
              ? 'Bàn phím cơ, chuột máy tính và thiết bị công nghệ.'
              : 'Mỹ phẩm, son môi và các thiết bị làm đẹp.',
          }));
          setTenants(normalized);

          // Restore active tenant from localStorage or fallback to first tenant in list
          const savedTenantId = localStorage.getItem('search_tenant_id');
          const found = normalized.find((t) => t.id === savedTenantId);
          if (found) {
            setActiveTenant(found);
          } else {
            setActiveTenant(normalized[0]);
          }
        }
      } catch (err) {
        console.error('Failed to load tenants dynamically, using offline fallbacks:', err);
      }
    };
    loadTenantsFromDB();
  }, []);

  const setActiveTenantById = (id: string) => {
    const tenant = tenants.find((t) => t.id === id);
    if (tenant) {
      setActiveTenant(tenant);
      localStorage.setItem('search_tenant_id', id);
    }
  };

  return (
    <TenantContext.Provider value={{ activeTenant, setActiveTenantById, tenants }}>
      {children}
    </TenantContext.Provider>
  );
};

export const useTenant = () => {
  const context = useContext(TenantContext);
  if (!context) {
    throw new Error('useTenant must be used within a TenantProvider');
  }
  return context;
};
