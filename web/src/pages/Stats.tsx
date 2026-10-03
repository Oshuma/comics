import { ListGroup, Nav } from 'react-bootstrap'
import { Link, NavLink } from 'react-router'
import { api, humanSize, pluralize } from '../api'
import Layout from '../components/Layout'
import { Icon, LoadError, Spinner, useLoad } from '../components/Common'

export default function Stats() {
  const { data, error } = useLoad(api.stats, [])

  const nav = (
    <Nav.Link as={NavLink} to="/stats">
      <Icon name="bar-chart" /> Stats
    </Nav.Link>
  )

  return (
    <Layout nav={nav} title="Stats">
      <div className="row justify-content-center">
        <div className="col-md-8 col-lg-6">
          {error ? (
            <LoadError error={error} />
          ) : !data ? (
            <Spinner />
          ) : (
            <ListGroup>
              <ListGroup.Item className="d-flex justify-content-between">
                <strong>Total</strong>
                <span>{humanSize(data.totalSize)}</span>
              </ListGroup.Item>
              {data.groups.map((g) => (
                <ListGroup.Item key={g.id} className="d-flex justify-content-between gap-2">
                  <span className="min-w-0 text-truncate">
                    <Link to={`/groups/${g.id}`}>{g.name}</Link> / {pluralize(g.comicCount, 'Comic')}
                  </span>
                  <span className="text-nowrap">{humanSize(g.diskSize)}</span>
                </ListGroup.Item>
              ))}
            </ListGroup>
          )}
        </div>
      </div>
    </Layout>
  )
}
