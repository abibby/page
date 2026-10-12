import { Link, NavLink, useLocation } from "react-router";
import { useEffect, useRef, type PropsWithChildren } from "react";
import { Icon } from "./icon";

export function Layout({ children }: PropsWithChildren) {
  const location = useLocation();
  const main = useRef<HTMLElement>(null);
  useEffect(() => {
    window.scrollTo(0, 0);
    main.current?.focus({ preventScroll: true });
  }, [location.pathname]);
  const type = new URLSearchParams(location.search).get("type");
  const inLibrary = location.pathname === "/";
  const section = location.pathname.includes("search")
    ? "Discover"
    : location.pathname.includes("downloads")
      ? "Downloads"
      : "Library";
  return (
    <div className="app-shell">
      <a className="skip-link" href="#main">
        Skip to content
      </a>
      <aside className="sidebar">
        <Link to="/" className="brand" aria-label="Page home">
          <span className="brand-mark">
            <Icon name="book" size={25} />
          </span>
          page<span className="brand-dot">.</span>
        </Link>
        <div className="sidebar-caption">A HOME FOR YOUR BOOKS</div>
        <nav aria-label="Main navigation" className="main-nav">
          <span className="nav-label">WORKSPACE</span>
          <Link
            to="/"
            className={`nav-item ${inLibrary && !type ? "active" : ""}`}
            aria-current={inLibrary && !type ? "page" : undefined}
          >
            <Icon name="library" />
            My library
          </Link>
          <NavLink to="/book/search" className="nav-item">
            <Icon name="compass" />
            Discover
          </NavLink>
          <NavLink to="/downloads" className="nav-item">
            <Icon name="download" />
            Downloads
          </NavLink>
          <span className="nav-label collection-label">COLLECTIONS</span>
          <Link
            to="/?type=ebooks"
            className={`nav-item ${inLibrary && type === "ebooks" ? "active" : ""}`}
            aria-current={inLibrary && type === "ebooks" ? "page" : undefined}
          >
            <Icon name="book" />
            Ebooks
          </Link>
          <Link
            to="/?type=audio"
            className={`nav-item ${inLibrary && type === "audio" ? "active" : ""}`}
            aria-current={inLibrary && type === "audio" ? "page" : undefined}
          >
            <Icon name="headphones" />
            Audiobooks
          </Link>
        </nav>
        <div className="sidebar-note">
          <span className="note-rule" />
          <Icon name="book" size={27} />
          <p>
            A good book is
            <br />
            always a good idea.
          </p>
          <span>Your next chapter awaits.</span>
        </div>
        <div className="sidebar-footer">
          <span className="personal-avatar">P</span>
          <div>
            Personal library<small>Made for your shelf</small>
          </div>
        </div>
      </aside>
      <div className="workspace">
        <header className="topbar">
          <span>
            <span className="topbar-home">Your workspace</span>
            <Icon name="chevron" size={14} />
            {section}
          </span>
          <Link to="/book/search" className="topbar-link">
            Find your next book <Icon name="plus" size={17} />
          </Link>
        </header>
        <main ref={main} id="main" className="main-content" tabIndex={-1}>
          {children}
        </main>
        <footer className="page-footer">
          <span>YOUR BOOKS. YOUR OWN LITTLE WORLD.</span>
          <span>
            Made for the love of reading <Icon name="book" size={14} />
          </span>
        </footer>
      </div>
    </div>
  );
}
