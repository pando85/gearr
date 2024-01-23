import React, { useState } from 'react';
import { Link } from 'react-router-dom';
import {
  Collapse,
  Navbar,
  NavbarToggler,
  Nav,
  NavItem,
  NavLink,
} from 'reactstrap';
import { ThemeToggle } from './Theme';
import GitHubIcon from '@mui/icons-material/GitHub';

interface CollapseNavProps {
  isOpen: boolean;
  className?: string;
  id?: string;
}

const CollapseNav: React.FC<CollapseNavProps> = ({ isOpen, className, id }) => (
  <Collapse className={className} id={id} isOpen={isOpen} navbar style={{ justifyContent: 'space-between' }}>
    <Nav className="ml-0" navbar>
      <NavItem>
        <NavLink tag={Link} to="/jobs">
          Jobs
        </NavLink>
      </NavItem>
      <NavItem>
        <NavLink href="https://github.com/pando85/transcoder" title="GitHub">
          <GitHubIcon />
        </NavLink>
      </NavItem>
    </Nav>
  </Collapse>
);

const Navigation: React.FC = () => {
  const [isOpen, setIsOpen] = useState(false);
  const toggle = () => setIsOpen(!isOpen);

  return (
    <Navbar className="top-bar" dark color="dark" expand="md" fixed="top">
      <NavbarToggler onClick={toggle} className="mr-2" />
      <Link className="link pt-0 navbar-brand" to='/jobs'>
        <img src="/logo.svg" alt="Transcoder" className="d-inline-block align-top logo" />
        <span className="d-none d-sm-inline">Transcoder</span>
      </Link>
      <CollapseNav isOpen={isOpen} className="d-none d-sm-inline" />
      <ThemeToggle />
      <CollapseNav isOpen={isOpen} className="high-level" id="large" />
    </Navbar>
  );
};

export default Navigation;
