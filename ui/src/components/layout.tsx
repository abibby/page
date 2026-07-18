import { Link } from "react-router";
import { routePath } from "../routes";
import type { PropsWithChildren } from "react";
import styles from './layout.module.css';

export function Layout(props: PropsWithChildren) {
  return (
    <>
      <nav className={styles.nav}>
        <Link to={routePath("home")}>Home</Link>
        <Link to={routePath("book.search")}>Add Book</Link>
      </nav>
      {props.children}
    </>
  );
}
