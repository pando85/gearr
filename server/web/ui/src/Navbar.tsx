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

const Navigation: React.FC = () => {
  const [isOpen, setIsOpen] = useState(false);
  const toggle = () => setIsOpen(!isOpen);

  return (
    <Navbar className="mb-3" dark color="dark" expand="md" fixed="top">
      <NavbarToggler onClick={toggle} className="mr-2" />
      <Link className="link pt-0 navbar-brand" to='/jobs'>
        <img src="/logo.svg" alt="Transcoder" className="d-inline-block align-top logo" />
        Transcoder
      </Link>
      <Collapse isOpen={isOpen} navbar style={{ justifyContent: 'space-between' }}>
        <Nav className="ml-0" navbar>
          <NavItem>
            <NavLink tag={Link} to="/jobs">
              Jobs
            </NavLink>
          </NavItem>
          <NavItem>
            <NavLink href="https://github.com/pando85/transcoder" title="GitHub"><GitHubIcon /></NavLink>
          </NavItem>
        </Nav>
      </Collapse>
      <ThemeToggle />
    </Navbar>
  );
};

export default Navigation;
