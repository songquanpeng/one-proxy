import React from 'react';
import { Header, Segment } from 'semantic-ui-react';
import ProfilesTable from '../../components/ProfilesTable';

const Profile = () => (
  <>
    <Segment>
      <Header as='h3'>管理订阅</Header>
      <ProfilesTable />
    </Segment>
  </>
);

export default Profile;
