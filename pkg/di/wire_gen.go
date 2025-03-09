// Code generated by Wire. DO NOT EDIT.

//go:generate go run -mod=mod github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package di

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/cron"
	"github.com/GoSimplicity/AI-CloudOps/internal/job"
	api5 "github.com/GoSimplicity/AI-CloudOps/internal/k8s/api"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/admin"
	uesr "github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/uesr"
	admin2 "github.com/GoSimplicity/AI-CloudOps/internal/k8s/service/admin"
	user2  "github.com/GoSimplicity/AI-CloudOps/internal/k8s/service/uesr"
	api4 "github.com/GoSimplicity/AI-CloudOps/internal/not_auth/api"
	service4 "github.com/GoSimplicity/AI-CloudOps/internal/not_auth/service"
	api6 "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/api"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/scrape"
	alert2 "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/alert"
	scrape2 "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/scrape"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/yaml"
	api2 "github.com/GoSimplicity/AI-CloudOps/internal/system/api"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/service"
	api3 "github.com/GoSimplicity/AI-CloudOps/internal/tree/api"
	dao3 "github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	service3 "github.com/GoSimplicity/AI-CloudOps/internal/tree/service"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/ssh"
	"github.com/GoSimplicity/AI-CloudOps/internal/user/api"
	dao2 "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	service2 "github.com/GoSimplicity/AI-CloudOps/internal/user/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
)

import (
	_ "github.com/google/wire"
)

// Injectors from wire.go:

func InitWebServer() *Cmd {
	cmdable := InitRedis()
	handler := utils.NewJWTHandler(cmdable)
	logger := InitLogger()
	db := InitDB()
	enforcer := InitCasbin(db)
	auditDAO := dao.NewAuditDAO(db, logger)
	auditService := service.NewAuditService(auditDAO)
	v := InitMiddlewares(handler, logger, enforcer, auditService)
	apiDAO := dao.NewApiDAO(db, enforcer, logger)
	permissionDAO := dao.NewPermissionDAO(db, logger, enforcer, apiDAO)
	userDAO := dao2.NewUserDAO(db, logger, permissionDAO)
	roleDAO := dao.NewRoleDAO(db, logger, enforcer, permissionDAO)
	roleService := service.NewRoleService(roleDAO, logger)
	permissionService := service.NewPermissionService(logger, permissionDAO)
	userService := service2.NewUserService(userDAO, roleService, permissionService)
	userHandler := api.NewUserHandler(userService, logger, handler)
	apiService := service.NewApiService(logger, apiDAO)
	apiHandler := api2.NewApiHandler(apiService)
	menuDAO := dao.NewMenuDAO(db, logger)
	menuService := service.NewMenuService(menuDAO, logger)
	menuHandler := api2.NewMenuHandler(menuService)
	roleHandler := api2.NewRoleHandler(roleService, apiService, permissionService, logger)
	permissionHandler := api2.NewPermissionHandler(permissionService)
	treeNodeDAO := dao3.NewTreeNodeDAO(db, logger)
	treeNodeService := service3.NewTreeNodeService(treeNodeDAO, userDAO, logger)
	treeNodeHandler := api3.NewTreeNodeHandler(treeNodeService)
	treeAliResourceDAO := dao3.NewAliResourceDAO(cmdable)
	treeEcsDAO := dao3.NewTreeEcsDAO(db, logger)
	treeEcsResourceDAO := dao3.NewEcsResourceDAO(logger, db)
	aliResourceService := service3.NewAliResourceService(logger, treeAliResourceDAO, cmdable, treeEcsDAO, treeEcsResourceDAO)
	aliResourceHandler := api3.NewAliResourceHandler(aliResourceService)
	ecsResourceService := service3.NewEcsResourceService(logger, treeEcsResourceDAO, treeEcsDAO)
	ecsResourceHandler := api3.NewEcsResourceHandler(ecsResourceService)
	ecsService := service3.NewEcsService(logger, treeEcsDAO, treeNodeDAO)
	ecsSSH := ssh.NewSSH(logger)
	ecsHandler := api3.NewEcsHandler(ecsService, logger, ecsSSH)
	treeElbDAO := dao3.NewTreeElbDAO(db, logger)
	elbService := service3.NewElbService(logger, treeElbDAO, treeNodeDAO)
	elbHandler := api3.NewElbHandler(elbService)
	treeRdsDAO := dao3.NewTreeRdsDAO(db, logger)
	rdsService := service3.NewRdsService(logger, treeRdsDAO, treeNodeDAO)
	rdsHandler := api3.NewRdsHandler(rdsService)
	notAuthService := service4.NewNotAuthService(logger, treeNodeDAO)
	notAuthHandler := api4.NewNotAuthHandler(notAuthService)
	clusterDAO := admin.NewClusterDAO(db, logger)
	k8sClient := client.NewK8sClient(logger, clusterDAO)
	asynqClient := InitAsynqClient()
	clusterService := admin2.NewClusterService(clusterDAO, k8sClient, logger, asynqClient)
	k8sClusterHandler := api5.NewK8sClusterHandler(logger, clusterService)
	configMapService := admin2.NewConfigMapService(clusterDAO, k8sClient, logger)
	k8sConfigMapHandler := api5.NewK8sConfigMapHandler(logger, configMapService)
	deploymentService := admin2.NewDeploymentService(clusterDAO, k8sClient, logger)
	k8sDeploymentHandler := api5.NewK8sDeploymentHandler(logger, deploymentService)
	namespaceService := admin2.NewNamespaceService(clusterDAO, k8sClient, logger)
	k8sNamespaceHandler := api5.NewK8sNamespaceHandler(logger, namespaceService)
	nodeService := admin2.NewNodeService(clusterDAO, k8sClient, logger)
	k8sNodeHandler := api5.NewK8sNodeHandler(logger, nodeService)
	podService := admin2.NewPodService(clusterDAO, k8sClient, logger)
	k8sPodHandler := api5.NewK8sPodHandler(logger, podService)
	svcService := admin2.NewSvcService(clusterDAO, k8sClient, logger)
	k8sSvcHandler := api5.NewK8sSvcHandler(logger, svcService)
	taintService := admin2.NewTaintService(clusterDAO, k8sClient, logger)
	k8sTaintHandler := api5.NewK8sTaintHandler(logger, taintService)
	yamlTaskDAO := admin.NewYamlTaskDAO(db, logger)
	yamlTemplateDAO := admin.NewYamlTemplateDAO(db, logger)
	yamlTaskService := admin2.NewYamlTaskService(yamlTaskDAO, clusterDAO, yamlTemplateDAO, k8sClient, logger)
	k8sYamlTaskHandler := api5.NewK8sYamlTaskHandler(logger, yamlTaskService)
	yamlTemplateService := admin2.NewYamlTemplateService(yamlTemplateDAO, yamlTaskDAO, k8sClient, logger)
	k8sYamlTemplateHandler := api5.NewK8sYamlTemplateHandler(logger, yamlTemplateService)
	k8sAppDao:=uesr.NewAppDAO(db, logger)
	k8sInstanceDao:=uesr.NewInstanceDAO(db,logger)
	k8sProjectDao:=uesr.NewProjectDAO(db, logger)
	K8sInstanceService:=user2.NewInstanceService(clusterDAO,k8sInstanceDao, k8sClient, logger)
	k8sAppService := user2.NewAppService(clusterDAO,k8sAppDao,k8sInstanceDao, k8sClient, logger)
	k8sProjectService := user2.NewProjectService(clusterDAO,k8sProjectDao,k8sAppDao,k8sInstanceDao, k8sClient, logger)
	K8sCoreService := user2.NewCronjobService( k8sClient, logger)
	k8sAppHandler := api5.NewK8sAppHandler(logger,K8sInstanceService, k8sAppService,k8sProjectService,K8sCoreService)
	alertManagerEventDAO := alert.NewAlertManagerEventDAO(db, logger, userDAO)
	scrapePoolDAO := scrape.NewScrapePoolDAO(db, logger, userDAO)
	scrapeJobDAO := scrape.NewScrapeJobDAO(db, logger, userDAO)
	promConfigCache := cache.NewPromConfigCache(logger, scrapePoolDAO, scrapeJobDAO)
	alertManagerPoolDAO := alert.NewAlertManagerPoolDAO(db, logger, userDAO)
	alertManagerSendDAO := alert.NewAlertManagerSendDAO(db, logger, userDAO)
	alertConfigCache := cache.NewAlertConfigCache(logger, alertManagerPoolDAO, alertManagerSendDAO)
	alertManagerRuleDAO := alert.NewAlertManagerRuleDAO(db, logger, userDAO)
	ruleConfigCache := cache.NewRuleConfigCache(logger, scrapePoolDAO, alertManagerRuleDAO)
	alertManagerRecordDAO := alert.NewAlertManagerRecordDAO(db, logger, userDAO)
	recordConfigCache := cache.NewRecordConfig(logger, scrapePoolDAO, alertManagerRecordDAO)
	monitorCache := cache.NewMonitorCache(promConfigCache, alertConfigCache, ruleConfigCache, recordConfigCache, logger)
	alertManagerEventService := alert2.NewAlertManagerEventService(alertManagerEventDAO, monitorCache, logger, userDAO, alertManagerSendDAO)
	alertEventHandler := api6.NewAlertEventHandler(logger, alertManagerEventService)
	alertManagerPoolService := alert2.NewAlertManagerPoolService(alertManagerPoolDAO, alertManagerSendDAO, monitorCache, logger, userDAO)
	alertPoolHandler := api6.NewAlertPoolHandler(logger, alertManagerPoolService)
	alertManagerRuleService := alert2.NewAlertManagerRuleService(alertManagerRuleDAO, monitorCache, logger, userDAO)
	alertRuleHandler := api6.NewAlertRuleHandler(logger, alertManagerRuleService)
	configYamlService := yaml.NewPrometheusConfigService(promConfigCache, alertConfigCache, ruleConfigCache, recordConfigCache)
	configYamlHandler := api6.NewConfigYamlHandler(logger, configYamlService)
	alertManagerOnDutyDAO := alert.NewAlertManagerOnDutyDAO(db, logger, userDAO)
	alertManagerOnDutyService := alert2.NewAlertManagerOnDutyService(alertManagerOnDutyDAO, alertManagerSendDAO, monitorCache, logger, userDAO)
	onDutyGroupHandler := api6.NewOnDutyGroupHandler(logger, alertManagerOnDutyService)
	alertManagerRecordService := alert2.NewAlertManagerRecordService(alertManagerRecordDAO, scrapePoolDAO, monitorCache, logger, userDAO)
	recordRuleHandler := api6.NewRecordRuleHandler(logger, alertManagerRecordService)
	scrapePoolService := scrape2.NewPrometheusPoolService(scrapePoolDAO, monitorCache, logger, userDAO, scrapeJobDAO)
	scrapePoolHandler := api6.NewScrapePoolHandler(logger, scrapePoolService)
	scrapeJobService := scrape2.NewPrometheusScrapeService(scrapeJobDAO, monitorCache, logger, userDAO, treeNodeDAO)
	scrapeJobHandler := api6.NewScrapeJobHandler(logger, scrapeJobService)
	alertManagerSendService := alert2.NewAlertManagerSendService(alertManagerSendDAO, alertManagerRuleDAO, monitorCache, logger, userDAO)
	sendGroupHandler := api6.NewSendGroupHandler(logger, alertManagerSendService)
	auditHandler := api2.NewAuditHandler(auditService)
	engine := InitGinServer(v, userHandler, apiHandler, menuHandler, roleHandler, permissionHandler, treeNodeHandler, aliResourceHandler, ecsResourceHandler, ecsHandler, elbHandler, rdsHandler, notAuthHandler, k8sClusterHandler, k8sConfigMapHandler, k8sDeploymentHandler, k8sNamespaceHandler, k8sNodeHandler, k8sPodHandler, k8sSvcHandler, k8sTaintHandler, k8sYamlTaskHandler, k8sYamlTemplateHandler, k8sAppHandler, alertEventHandler, alertPoolHandler, alertRuleHandler, configYamlHandler, onDutyGroupHandler, recordRuleHandler, scrapePoolHandler, scrapeJobHandler, sendGroupHandler, auditHandler)
	createK8sClusterTask := job.NewCreateK8sClusterTask(logger, k8sClient, clusterDAO)
	updateK8sClusterTask := job.NewUpdateK8sClusterTask(logger, k8sClient, clusterDAO)
	cronManager := cron.NewCronManager(logger, alertManagerOnDutyDAO, treeEcsDAO, clusterDAO, k8sClient)
	timedTask := job.NewTimedTask(logger, k8sClient, monitorCache, cronManager)
	routes := job.NewRoutes(createK8sClusterTask, updateK8sClusterTask, timedTask)
	server := InitAsynqServer()
	scheduler := InitScheduler()
	timedScheduler := job.NewTimedScheduler(scheduler)
	cmd := &Cmd{
		Server:    engine,
		Start:     aliResourceService,
		Routes:    routes,
		Asynq:     server,
		Scheduler: timedScheduler,
	}
	return cmd
}
