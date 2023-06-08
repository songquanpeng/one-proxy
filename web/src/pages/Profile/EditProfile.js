import React, { useEffect, useState } from 'react';
import { Button, Form, Header, Segment } from 'semantic-ui-react';
import { useParams } from 'react-router-dom';
import { API, showError, showSuccess } from '../../helpers';

const EditProfile = () => {
  const params = useParams();
  const profileId = params.id;
  const isEdit = profileId !== undefined;
  const [loading, setLoading] = useState(isEdit);
  const originInputs = {
    name: '',
    description: '',
    url: ''
  };
  const [inputs, setInputs] = useState(originInputs);
  const { name, description, url } = inputs;
  const handleInputChange = (e, { name, value }) => {
    setInputs((inputs) => ({ ...inputs, [name]: value }));
  };

  const loadProfile = async () => {
    let res = await API.get(`/api/profile/${profileId}`);
    const { success, message, data } = res.data;
    if (success) {
      data.password = '';
      setInputs(data);
    } else {
      showError(message);
    }
    setLoading(false);
  };
  useEffect(() => {
    if (isEdit) {
      loadProfile().then();
    }
  }, []);

  const submit = async () => {
    let res = undefined;
    if (isEdit) {
      res = await API.put(`/api/profile/`, { ...inputs, id: parseInt(profileId) });
    } else {
      res = await API.post(`/api/profile`, inputs);
    }
    const { success, message } = res.data;
    if (success) {
      showSuccess('订阅更新成功！');
    } else {
      showError(message);
    }
  };

  return (
    <>
      <Segment loading={loading}>
        <Header as='h3'>{isEdit ? "更新订阅" : "新增订阅"}</Header>
        <Form autoComplete='new-password'>
          <Form.Field>
            <Form.Input
              label='名称'
              name='name'
              placeholder={'请输入名称'}
              onChange={handleInputChange}
              value={name}
            />
          </Form.Field>
          <Form.Field>
            <Form.Input
              label='描述'
              name='description'
              placeholder={'请输入描述信息'}
              onChange={handleInputChange}
              value={description}
            />
          </Form.Field>
          <Form.Field>
            <Form.Input
              label='订阅链接'
              name='url'
              type={'url'}
              placeholder={'请输入订阅链接'}
              onChange={handleInputChange}
              value={url}
            />
          </Form.Field>
          <Button onClick={submit}>提交</Button>
        </Form>
      </Segment>
    </>
  );
};

export default EditProfile;
