import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import Storefront from './pages/Storefront';
import Admin from './pages/Admin';
import { TenantProvider } from './context/TenantContext';

function App() {
  return (
    <TenantProvider>
      <Router>
        <Routes>
          <Route path="/" element={<Storefront />} />
          <Route path="/admin" element={<Admin />} />
          <Route path="*" element={<Storefront />} />
        </Routes>
      </Router>
    </TenantProvider>
  );
}

export default App;
