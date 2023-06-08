import React, { useEffect, useState } from 'react';
import { Button, Divider, Form, Grid, Header, Modal } from 'semantic-ui-react';
import { API, showError, showSuccess } from '../helpers';
import { marked } from 'marked';

const OtherSetting = () => {
  let [inputs, setInputs] = useState({
    Footer: '',
    Notice: '',
    About: '',
    SystemName: '',
    HomePageLink: '',
  });
  let [loading, setLoading] = useState(false);
  const [showUpdateModal, setShowUpdateModal] = useState(false);
  const [updateData, setUpdateData] = useState({
    tag_name: '',
    content: '',
  });

  const getOptions = async () => {
    const res = await API.get('/api/option/');
    const { success, message, data } = res.data;
    if (success) {
      let newInputs = {};
      data.forEach((item) => {
        if (item.key in inputs) {
          newInputs[item.key] = item.value;
        }
      });
      setInputs(newInputs);
    } else {
      showError(message);
    }
  };

  useEffect(() => {
    getOptions().then();
  }, []);

  const updateOption = async (key, value) => {
    setLoading(true);
    const res = await API.put('/api/option', {
      key,
      value,
    });
    const { success, message } = res.data;
    if (success) {
      setInputs((inputs) => ({ ...inputs, [key]: value }));
    } else {
      showError(message);
    }
    setLoading(false);
  };

  const handleInputChange = async (e, { name, value }) => {
    setInputs((inputs) => ({ ...inputs, [name]: value }));
  };

  const submitNotice = async () => {
    await updateOption('Notice', inputs.Notice);
  };

  const submitFooter = async () => {
    await updateOption('Footer', inputs.Footer);
  };

  const submitSystemName = async () => {
    await updateOption('SystemName', inputs.SystemName);
  };

  const submitHomePageLink = async () => {
    await updateOption('HomePageLink', inputs.HomePageLink);
  };

  const submitAbout = async () => {
    await updateOption('About', inputs.About);
  };

  const openGitHubRelease = () => {
    window.location =
      'https://github.com/songquanpeng/gin-template/releases/latest';
  };

  const checkUpdate = async () => {
    const res = await API.get(
      'https://api.github.com/repos/songquanpeng/gin-template/releases/latest'
    );
    const { tag_name, body } = res.data;
    if (tag_name === process.env.REACT_APP_VERSION) {
      showSuccess(`已是最新版本：${tag_name}`);
    } else {
      setUpdateData({
        tag_name: tag_name,
        content: marked.parse(body),
      });
      setShowUpdateModal(true);
    }
  };

  return (
    <Grid columns={1}>
      <Grid.Column>
        <Form loading={loading}>
          <Header as='h3'>通用设置</Header>
          <Form.Button onClick={checkUpdate}>检查更新</Form.Button>
          <Form.Group widths='equal'>
            <Form.TextArea
              label='公告'
              placeholder='在此输入新的公告内容'
              value={inputs.Notice}
              name='Notice'
              onChange={handleInputChange}
              style={{ minHeight: 150, fontFamily: 'JetBrains Mono, Consolas' }}
            />
          </Form.Group>
          <Form.Button onClick={submitNotice}>保存公告</Form.Button>
          <Divider />
          <Header as='h3'>个性化设置</Header>
          <Form.Group widths='equal'>
            <Form.Input
              label='系统名称'
              placeholder='在此输入系统名称'
              value={inputs.SystemName}
              name='SystemName'
              onChange={handleInputChange}
            />
          </Form.Group>
          <Form.Button onClick={submitSystemName}>设置系统名称</Form.Button>
          <Form.Group widths='equal'>
            <Form.Input
              label='首页链接'
              placeholder='在此输入首页链接，设置后将通过 iframe 方式嵌入该网页'
              value={inputs.HomePageLink}
              name='HomePageLink'
              onChange={handleInputChange}
              type='url'
            />
          </Form.Group>
          <Form.Button onClick={submitHomePageLink}>设置首页链接</Form.Button>
          <Form.Group widths='equal'>
            <Form.TextArea
              label='关于'
              placeholder='在此输入新的关于内容，支持 Markdown & HTML 代码'
              value={inputs.About}
              name='About'
              onChange={handleInputChange}
              style={{ minHeight: 150, fontFamily: 'JetBrains Mono, Consolas' }}
            />
          </Form.Group>
          <Form.Button onClick={submitAbout}>保存关于</Form.Button>
          <Form.Group widths='equal'>
            <Form.Input
              label='页脚'
              placeholder='在此输入新的页脚，留空则使用默认页脚，支持 HTML 代码'
              value={inputs.Footer}
              name='Footer'
              onChange={handleInputChange}
            />
          </Form.Group>
          <Form.Button onClick={submitFooter}>设置页脚</Form.Button>
        </Form>
      </Grid.Column>
      <Modal
        onClose={() => setShowUpdateModal(false)}
        onOpen={() => setShowUpdateModal(true)}
        open={showUpdateModal}
      >
        <Modal.Header>新版本：{updateData.tag_name}</Modal.Header>
        <Modal.Content>
          <Modal.Description>
            <div dangerouslySetInnerHTML={{ __html: updateData.content }}></div>
          </Modal.Description>
        </Modal.Content>
        <Modal.Actions>
          <Button onClick={() => setShowUpdateModal(false)}>关闭</Button>
          <Button
            content='详情'
            onClick={() => {
              setShowUpdateModal(false);
              openGitHubRelease();
            }}
          />
        </Modal.Actions>
      </Modal>
    </Grid>
  );
};

export default OtherSetting;
