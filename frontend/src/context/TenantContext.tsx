import React, { createContext, useContext, useState, useEffect } from 'react';

export interface Tenant {
  id: string;
  name: string;
  description: string;
}

export const TENANTS: Tenant[] = [
  {
    id: 'd3b07384-d113-4956-a5db-251d50c18d01',
    name: 'Tenant A (PC & Bàn Phím)',
    description: 'Bàn phím cơ, chuột máy tính và thiết bị công nghệ.',
  },
  {
    id: 'a1a2a3a4-b1b2-c1c2-d1d2-e1e2e3e4e5e6',
    name: 'Tenant B (Mỹ phẩm & Chăm sóc)',
    description: 'Son môi, kem dưỡng da và mỹ phẩm chăm sóc sắc đẹp.',
  },
];

interface TenantContextType {
  activeTenant: Tenant;
  setActiveTenantById: (id: string) => void;
  tenants: Tenant[];
}

const TenantContext = createContext<TenantContextType | undefined>(undefined);

export const TenantProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [activeTenant, setActiveTenant] = useState<Tenant>(TENANTS[0]);

  useEffect(() => {
    const savedTenantId = localStorage.getItem('search_tenant_id');
    if (savedTenantId) {
      const tenant = TENANTS.find((t) => t.id === savedTenantId);
      if (tenant) {
        setActiveTenant(tenant);
      }
    }
  }, []);

  const setActiveTenantById = (id: string) => {
    const tenant = TENANTS.find((t) => t.id === id);
    if (tenant) {
      setActiveTenant(tenant);
      localStorage.setItem('search_tenant_id', id);
    }
  };

  return (
    <TenantContext.Provider value={{ activeTenant, setActiveTenantById, tenants: TENANTS }}>
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
