import React, { useEffect, useState } from 'react';
import { Button, Form, Label, Pagination, Popup, Table } from 'semantic-ui-react';
import { Link } from 'react-router-dom';
import { API, copy, showError, showSuccess, showWarning, timestamp2string } from '../helpers';

import { ITEMS_PER_PAGE } from '../constants';

function renderTimestamp(timestamp) {
  return (
    <>
      {timestamp2string(timestamp)}
    </>
  );
}

const ProfilesTable = () => {
  const [profiles, setProfiles] = useState([]);
  const [loading, setLoading] = useState(true);
  const [activePage, setActivePage] = useState(1);
  const [searchKeyword, setSearchKeyword] = useState('');
  const [searching, setSearching] = useState(false);

  const loadProfiles = async (startIdx) => {
    const res = await API.get(`/api/profile/?p=${startIdx}`);
    const { success, message, data } = res.data;
    if (success) {
      if (startIdx === 0) {
        setProfiles(data);
      } else {
        let newProfiles = profiles;
        newProfiles.push(...data);
        setProfiles(newProfiles);
      }
    } else {
      showError(message);
    }
    setLoading(false);
  };

  const onPaginationChange = (e, { activePage }) => {
    (async () => {
      if (activePage === Math.ceil(profiles.length / ITEMS_PER_PAGE) + 1) {
        // In this case we have to load more data and then append them.
        await loadProfiles(activePage - 1);
      }
      setActivePage(activePage);
    })();
  };

  useEffect(() => {
    loadProfiles(0)
      .then()
      .catch((reason) => {
        showError(reason);
      });
  }, []);

  const manageProfile = async (id, action, idx) => {
    let res;
    switch (action) {
      case 'delete':
        res = await API.delete(`/api/profile/${id}`);
        break;
      case 'reset':
        res = await API.get(`/api/profile/reset/${id}`);
        break;
      case 'enable':
      case 'disable':
        res = await API.put(`/api/profile?status_only=true`, {
          id,
          status: action === 'enable' ? 1 : 2
        });
        break;
    }
    const { success, message, data } = res.data;
    if (success) {
      showSuccess('操作成功完成！');
      let newProfiles = [...profiles];
      let realIdx = (activePage - 1) * ITEMS_PER_PAGE + idx;
      switch (action) {
        case 'reset':
          newProfiles[realIdx].token = data;
          break;
        case 'delete':
          newProfiles[realIdx].deleted = true;
          break;
        default:
          newProfiles[realIdx].status = action === 'enable' ? 1 : 2;
          break;
      }
      setProfiles(newProfiles);
    } else {
      showError(message);
    }
  };

  const renderStatus = (status) => {
    switch (status) {
      case 1:
        return <Label basic>已启用</Label>;
      case 2:
        return (
          <Label basic color='red'>
            已禁用
          </Label>
        );
      default:
        return (
          <Label basic color='grey'>
            未知状态
          </Label>
        );
    }
  };

  const searchProfiles = async () => {
    if (searchKeyword === '') {
      // if keyword is blank, load files instead.
      await loadProfiles(0);
      setActivePage(1);
      return;
    }
    setSearching(true);
    const res = await API.get(`/api/profile/search?keyword=${searchKeyword}`);
    const { success, message, data } = res.data;
    if (success) {
      setProfiles(data);
      setActivePage(1);
    } else {
      showError(message);
    }
    setSearching(false);
  };

  const handleKeywordChange = async (e, { value }) => {
    setSearchKeyword(value.trim());
  };

  const sortProfile = (key) => {
    if (profiles.length === 0) return;
    setLoading(true);
    let sortedProfiles = [...profiles];
    sortedProfiles.sort((a, b) => {
      return ('' + a[key]).localeCompare(b[key]);
    });
    if (sortedProfiles[0].id === profiles[0].id) {
      sortedProfiles.reverse();
    }
    setProfiles(sortedProfiles);
    setLoading(false);
  };

  return (
    <>
      <Form onSubmit={searchProfiles}>
        <Form.Input
          icon='search'
          fluid
          iconPosition='left'
          placeholder='搜索订阅名称 ...'
          value={searchKeyword}
          loading={searching}
          onChange={handleKeywordChange}
        />
      </Form>

      <Table basic>
        <Table.Header>
          <Table.Row>
            <Table.HeaderCell
              style={{ cursor: 'pointer' }}
              onClick={() => {
                sortProfile('name');
              }}
            >
              名称
            </Table.HeaderCell>
            <Table.HeaderCell
              style={{ cursor: 'pointer' }}
              onClick={() => {
                sortProfile('description');
              }}
            >
              描述
            </Table.HeaderCell>
            <Table.HeaderCell
              style={{ cursor: 'pointer' }}
              onClick={() => {
                sortProfile('created_time');
              }}
            >
              创建时间
            </Table.HeaderCell>
            <Table.HeaderCell
              style={{ cursor: 'pointer' }}
              onClick={() => {
                sortProfile('status');
              }}
            >
              状态
            </Table.HeaderCell>
            <Table.HeaderCell>操作</Table.HeaderCell>
          </Table.Row>
        </Table.Header>

        <Table.Body>
          {profiles
            .slice(
              (activePage - 1) * ITEMS_PER_PAGE,
              activePage * ITEMS_PER_PAGE
            )
            .map((profile, idx) => {
              if (profile.deleted) return <></>;
              return (
                <Table.Row key={profile.id}>
                  <Table.Cell>{profile.name ? profile.name : "无名称"}</Table.Cell>
                  <Table.Cell>{profile.description ? profile.description : "无描述信息"}</Table.Cell>
                  <Table.Cell>{renderTimestamp(profile.created_time)}</Table.Cell>
                  <Table.Cell>{renderStatus(profile.status)}</Table.Cell>
                  <Table.Cell>
                    <div>
                      <Button
                        size={'small'}
                        positive
                        onClick={async () => {
                          let link = `${window.location.protocol}//${window.location.host}/profile/${profile.token}`;
                          if (await copy(link)) {
                            showSuccess('已复制到剪贴板！');
                          } else {
                            showWarning('无法复制到剪贴板，请手动复制，已将令牌填入搜索框。');
                            setSearchKeyword(link);
                          }
                        }}
                      >
                        复制
                      </Button>
                      <Button
                        size={'small'}
                        color={'yellow'}
                        onClick={() => {
                          manageProfile(profile.id, 'reset', idx).then();
                        }}
                      >
                        重置
                      </Button>
                      <Popup
                        trigger={
                          <Button size='small' negative>
                            删除
                          </Button>
                        }
                        on='click'
                        flowing
                        hoverable
                      >
                        <Button
                          negative
                          onClick={() => {
                            manageProfile(profile.id, 'delete', idx);
                          }}
                        >
                          删除订阅 {profile.name}
                        </Button>
                      </Popup>
                      <Button
                        size={'small'}
                        onClick={() => {
                          manageProfile(
                            profile.id,
                            profile.status === 1 ? 'disable' : 'enable',
                            idx
                          );
                        }}
                      >
                        {profile.status === 1 ? '禁用' : '启用'}
                      </Button>
                      <Button
                        size={'small'}
                        as={Link}
                        to={'/profile/edit/' + profile.id}
                      >
                        编辑
                      </Button>
                    </div>
                  </Table.Cell>
                </Table.Row>
              );
            })}
        </Table.Body>

        <Table.Footer>
          <Table.Row>
            <Table.HeaderCell colSpan='6'>
              <Button size='small' as={Link} to='/profile/add' loading={loading}>
                添加新的订阅
              </Button>
              <Pagination
                floated='right'
                activePage={activePage}
                onPageChange={onPaginationChange}
                size='small'
                siblingRange={1}
                totalPages={
                  Math.ceil(profiles.length / ITEMS_PER_PAGE) +
                  (profiles.length % ITEMS_PER_PAGE === 0 ? 1 : 0)
                }
              />
            </Table.HeaderCell>
          </Table.Row>
        </Table.Footer>
      </Table>
    </>
  );
};

export default ProfilesTable;
