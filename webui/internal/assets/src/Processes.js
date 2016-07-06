import React from 'react';
import UnixTime from './UnixTime';
import Abbrev from './Abbrev';
import styles from './bootstrap.min.css';
import cx from './cx';

class BusyWorkers extends React.Component {
  static propTypes = {
    worker: React.PropTypes.arrayOf(React.PropTypes.object).isRequired,
  }

  render() {
    return (
      <div className={styles.tableResponsive}>
        <table className={styles.table}>
          <tbody>
            <tr>
              <th>Name</th>
              <th>Arguments</th>
              <th>Started At</th>
              <th>Check-in At</th>
              <th>Check-in</th>
            </tr>
            {
              this.props.worker.map((worker) => {
                return (
                  <tr key={worker.WorkerID}>
                    <td>{worker.JobName}</td>
                    <td>{worker.ArgsJSON}</td>
                    <td><UnixTime ts={worker.StartedAt}/></td>
                    <td><UnixTime ts={worker.CheckinAt}/></td>
                    <td>{worker.Checkin}</td>
                  </tr>
                  );
              })
            }
          </tbody>
        </table>
      </div>
    );
  }
}

export default class Processes extends React.Component {
  static propTypes = {
    busyWorkerURL: React.PropTypes.string,
    workerPoolURL: React.PropTypes.string,
  }

  state = {
    busyWorker: [],
    workerPool: []
  }

  componentWillMount() {
    if (this.props.busyWorkerURL) {
      fetch(this.props.busyWorkerURL).
        then((resp) => resp.json()).
        then((data) => {
          if (data) {
            this.setState({
              busyWorker: data
            });
          }
        });
    }
    if (this.props.workerPoolURL) {
      fetch(this.props.workerPoolURL).
        then((resp) => resp.json()).
        then((data) => {
          let workers = [];
          data.map((worker) => {
            if (worker.Host != '') {
              workers.push(worker);
            }
          });
          this.setState({
            workerPool: workers
          });
        });
    }
  }

  get workerCount() {
    let count = 0;
    this.state.workerPool.map((pool) => {
      count += pool.WorkerIDs.length;
    });
    return count;
  }

  getBusyPoolWorker(pool) {
    let workers = [];
    this.state.busyWorker.map((worker) => {
      if (pool.WorkerIDs.includes(worker.WorkerID)) {
        workers.push(worker);
      }
    });
    return workers;
  }

  render() {
    return (
      <section>
        <header>Processes</header>
        <p>{this.state.workerPool.length} Worker process(es). {this.state.busyWorker.length} active worker(s) out of {this.workerCount}.</p>
        {
          this.state.workerPool.map((pool) => {
            let busyWorker = this.getBusyPoolWorker(pool);
            return (
              <div key={pool.WorkerPoolID} className={cx(styles.panel, styles.panelDefault)}>
                <div className={styles.tableResponsive}>
                  <table className={styles.table}>
                    <tbody>
                      <tr>
                        <td>{pool.Host}: {pool.Pid}</td>
                        <td>Started <UnixTime ts={pool.StartedAt}/></td>
                        <td>Last Heartbeat <UnixTime ts={pool.HeartbeatAt}/></td>
                        <td>Concurrency {pool.Concurrency}</td>
                      </tr>
                      <tr>
                        <td colSpan="4">Servicing <Abbrev item={pool.JobNames} />.</td>
                      </tr>
                      <tr>
                        <td colSpan="4">{busyWorker.length} active worker(s) and {pool.WorkerIDs.length - busyWorker.length} idle.</td>
                      </tr>
                      <tr>
                        <td colSpan="4">
                          <div className={cx(styles.panel, styles.panelDefault)}>
                            <BusyWorkers worker={busyWorker} />
                          </div>
                        </td>
                      </tr>
                    </tbody>
                  </table>
                </div>
              </div>
              );
          })
        }
      </section>
    );
  }
}
