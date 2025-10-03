import { defineStore } from 'pinia';
import { login as apiLogin, logout as apiLogout } from '@/api/auth';
import { getInfo as apiGetInfo } from '@/api/user';
import { getToken, setToken, removeToken } from '@/utils/auth';
import router, { resetRouter } from '@/router';
import tagsViewStore from './tagsView';
import permissionStore from './permission';

export interface IUserState {
  token: string;
  userId: string,
  name: string;
  avatar: string;
  introduction: string;
  roles: string[];
}

interface LoginResponse {
  token: string;
}

interface UserProfile {
  roles: string[];
  nickname: string;
  avatar: string;
  introduction: string;
}

interface GetInfoResponse {
  user?: UserProfile;
}

export default defineStore({
  id: 'user',
  state: (): IUserState => ({
    token: getToken() || '',
    userId: '',
    name: '',
    avatar: '',
    introduction: '',
    roles: []
  }),
  getters: {},
  actions: {
    // user login
    login(userInfo): Promise<void> {
      const { username, password } = userInfo;
      return new Promise((resolve, reject) => {
        apiLogin({ username: username.trim(), password: password }).then((response: LoginResponse) => {
          const { token } = response;
          this.token = token;
          setToken(token);
          resolve();
        }).catch(error => {
          reject(error);
        });
      });
    },

    // get user info
    getInfo(): Promise<UserProfile> {
      return new Promise((resolve, reject) => {
        apiGetInfo().then((response: GetInfoResponse) => {
          const { user } = response;

          if (!user) {
            reject('Verification failed, please Login again.');
            return;
          }

          const { roles, nickname, avatar, introduction } = user;

          // roles must be a non-empty array
          if (!roles || roles.length <= 0) {
            reject('getInfo: roles must be a non-null array!');
            return;
          }

          this.roles = roles;
          this.name = nickname;
          this.avatar = avatar;
          this.introduction = introduction;
          resolve(user);
        }).catch(error => {
          reject(error);
        });
      });
    },

    // user logout
    logout(): Promise<void> {
      return new Promise((resolve, reject) => {
        apiLogout().then(() => {
          this.token = '';
          this.roles = [];
          removeToken();
          resetRouter();

          // reset visited views and cached views
          // to fixed https://github.com/PanJiaChen/vue-element-admin/issues/2485
          tagsViewStore().delAllViews();

          resolve();
        }).catch(error => {
          reject(error);
        });
      });
    },

    // remove token
    resetToken() {
      this.token = '';
      this.roles = [];
      removeToken();
    },

    // dynamically modify permissions
    async changeRoles(role: string): Promise<void> {
      const token = role + '-token';

      this.token = token;
      setToken(token);

      const infoRes = await this.getInfo();
      const roles = infoRes.roles || [];

      resetRouter();

      // generate accessible routes map based on roles
      const accessRoutes = await permissionStore().generateRoutes(roles);
      // dynamically add accessible routes
      // router.addRoutes(accessRoutes);
      accessRoutes.forEach(item => {
        router.addRoute(item);
      });

      // reset visited views and cached views
      tagsViewStore().delAllViews();
    }
  }
});
