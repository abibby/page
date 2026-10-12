import { isRouteErrorResponse, Link, useRouteError } from "react-router";
import { Layout } from "../components/layout";
import { BookSkeletons } from "../components/feedback";
import { Icon } from "../components/icon";

export function RouteError() {
  const error = useRouteError();
  const notFound = isRouteErrorResponse(error) && error.status === 404;
  return (
    <Layout>
      <div className="empty-state route-error">
        <span className="empty-icon">
          <Icon name="book" size={32} />
        </span>
        <span className="eyebrow">A PAGE OUT OF PLACE</span>
        <h1>
          {notFound ? "This shelf is empty." : "We couldn’t open this page."}
        </h1>
        <p>
          {notFound
            ? "The page or book you’re looking for could not be found."
            : error instanceof Error
              ? error.message
              : "Please try again in a moment."}
        </p>
        <div className="detail-actions">
          <Link className="button button-primary" to="/">
            Back to library
          </Link>
          <button
            className="button button-secondary"
            onClick={() => window.location.reload()}
          >
            Try again
          </button>
        </div>
      </div>
    </Layout>
  );
}

export function PageLoading() {
  return (
    <Layout>
      <p className="loading-message" role="status">
        Opening your library…
      </p>
      <BookSkeletons />
    </Layout>
  );
}
