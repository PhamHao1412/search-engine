import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import Home from './pages/Home';
import Storefront from './pages/Storefront';
import ProductDetail from './pages/ProductDetail';
import Admin from './pages/Admin';
import { TenantProvider } from './context/TenantContext';

function App() {
  return (
    <TenantProvider>
      <Router>
        <Routes>
          <Route path="/" element={<Home />} />
          <Route path="/search" element={<Storefront />} />
          <Route path="/products/:id" element={<ProductDetail />} />
          <Route path="/admin" element={<Admin />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </Router>
    </TenantProvider>
  );
}

export default App;
