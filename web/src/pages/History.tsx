import { Card, ListGroup, Nav } from 'react-bootstrap'
import { Link, NavLink, useSearchParams } from 'react-router'
import { api, type History } from '../api'
import Layout from '../components/Layout'
import { EntriesInfo, Icon, LoadError, Pagination, Spinner, useLoad } from '../components/Common'

function formatDate(iso: string) {
  const d = new Date(iso)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${pad(d.getMonth() + 1)}-${pad(d.getDate())}-${d.getFullYear()}`
}

function HistoryItem({ h }: { h: History }) {
  return (
    <ListGroup.Item className="d-flex justify-content-between gap-2">
      <span className="min-w-0 text-truncate">
        {h.groupId ? (
          <Link to={`/groups/${h.groupId}`}>{h.groupName}</Link>
        ) : (
          <Link to={`/upload?group_name=${encodeURIComponent(h.groupName)}`} title="Group was removed; upload into it">
            {h.groupName}
          </Link>
        )}
        {' / '}
        {h.comicId ? <Link to={`/comics/${h.comicId}`}>{h.comicName}</Link> : h.comicName}
      </span>
      <em className="text-body-secondary text-nowrap">{formatDate(h.createdAt)}</em>
    </ListGroup.Item>
  )
}

export default function HistoryPage() {
  const [params, setParams] = useSearchParams()
  const page = Math.max(1, Number(params.get('page')) || 1)
  const { data, error } = useLoad(() => api.history(page), [page])

  const nav = (
    <Nav.Link as={NavLink} to="/history">
      <Icon name="clock-history" /> History
    </Nav.Link>
  )

  return (
    <Layout nav={nav} title="History">
      <div className="row justify-content-center">
        <div className="col-md-8 col-lg-6">
          {error ? (
            <LoadError error={error} />
          ) : !data ? (
            <Spinner />
          ) : (
            <>
              <Card className="mb-3">
                <Card.Header>
                  <strong>History</strong>
                </Card.Header>
                {data.items.length > 0 ? (
                  <ListGroup variant="flush">
                    {data.items.map((h) => (
                      <HistoryItem key={h.id} h={h} />
                    ))}
                  </ListGroup>
                ) : (
                  <Card.Body>The comics you have already read will show up in this list.</Card.Body>
                )}
              </Card>
              <EntriesInfo page={data.page} total={data.total} perPage={data.perPage} />
              <Pagination
                page={data.page}
                total={data.total}
                perPage={data.perPage}
                onChange={(p) => setParams({ page: String(p) })}
              />
            </>
          )}
        </div>
      </div>
    </Layout>
  )
}
