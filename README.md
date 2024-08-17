<p style="text-align:center;" align="center"><a href="https://layer5.io/meshery">
<picture align="center">
  <source media="(prefers-color-scheme: dark)" srcset="https://raw.githubusercontent.com/meshery/meshery/master/.github/assets/images/readme/meshery-logo-light-text-side.svg"  width="70%" align="center" style="margin-bottom:20px;">
  <source media="(prefers-color-scheme: light)" srcset="https://raw.githubusercontent.com/meshery/meshery/master/.github/assets/images/readme/meshery-logo-dark-text-side.svg" width="70%" align="center" style="margin-bottom:20px;">
  <img alt="Shows an illustrated light mode meshery logo in light color mode and a dark mode meshery logo dark color mode." src="https://raw.githubusercontent.com/meshery/meshery/master/.github/assets/images/readme/meshery-logo-dark-text-side.svg" width="70%" align="center" style="margin-bottom:20px;">
</picture></a><br /><br /></p>

<div align="center">

[![Docker Pulls](https://img.shields.io/docker/pulls/layer5/meshery-operator.svg)](https://hub.docker.com/r/layer5/meshery-operator)
[![Go Report Card](https://goreportcard.com/badge/github.com/layer5io/meshery-operator)](https://goreportcard.com/report/github.com/layer5io/meshery-operator)
[![Build Status](https://github.com/meshery/meshery-operator/actions/workflows/build-and-release.yml/badge.svg)](https://github.com/meshery/meshery-operator/actions)
[![GitHub](https://img.shields.io/github/license/layer5io/meshery-operator.svg)](LICENSE)
[![codecov](https://codecov.io/gh/meshery/meshery-operator/branch/master/graph/badge.svg?token=TJZ2L4JHSA)](https://codecov.io/gh/meshery/meshery-operator)
[![Website](https://img.shields.io/website/https/layer5.io/meshery.svg)](https://layer5.io/meshery/)
[![Twitter Follow](https://img.shields.io/twitter/follow/layer5.svg?label=Follow&style=social)](https://twitter.com/intent/follow?screen_name=mesheryio)
[![Discuss Users](https://img.shields.io/discourse/users?server=https%3A%2F%2Fdiscuss.layer5.io)](https://discuss.layer5.io)
[![Slack](https://img.shields.io/badge/Slack-@layer5.svg?logo=slack)](http://slack.layer5.io)
[![CII Best Practices](https://bestpractices.coreinfrastructure.org/projects/3564/badge)](https://bestpractices.coreinfrastructure.org/projects/3564)

</div>
<br />
<p align="center">
A self-service engineering platform, <a href="https://meshery.io">Meshery</a>, is the open source, cloud native manager that enables the design and management of all Kubernetes-based infrastructure and applications. Among other features, as an extensible platform, Meshery offers visual and collaborative GitOps, freeing you from the chains of YAML while managing Kubernetes multi-cluster deployments.
</p>
<br />

<a href="https://meshery.io/community"><img alt="Layer5 Service Mesh Community" src="./img/readme/meshery-operator-dark.svg" style="margin:10px;" width="165px" align="left" /></a>
# Meshery Operator
Meshery Operator is the multi-service mesh operator and implementation of MeshSync.
<div>&nbsp;</div>
<div>&nbsp;</div>
<br />
<br />
<br />

<a href="https://github.com/meshery/meshsync"><img align="left" src="https://raw.githubusercontent.com/layer5io/meshsync/master/.github/readme/images/meshsync.svg"  width="165px" /></a>
# MeshSync
MeshSync (Service Mesh Synchronization), an event-driven, continuous synchronization routine performs the task of ensuring that the state of configuration and status of operation of any service mesh and of the Kubernetes environment are known to Meshery. MeshSync runs as a Kubernetes custom controller under the control of Meshery Operator.
<br />
<br />
<br />

<a href="https://docs.meshery.io/assets/img/architecture/meshery-operator-and-meshsync.svg"><img src="./img/readme/meshery-broker.png" style="margin:10px;" width="165px" align="left" /></a>
# Meshery Broker
<a href="https://docs.google.com/presentation/d/1SQMfyu5shjpGKlYONdVzOtd7UYTgLWBcgUvHMLCZ2tY/edit#slide=id.g10af036b3dd_0_0">Meshery Broker</a> is a custom Kubernetes controller that provides data streaming across independent components of Meshery whether those components are running inside or outside of the Kubernetes cluster.
<br />
<br />
<br />




## Contributing

Please do! We're a warm and welcoming community of open source contributors. Please join. All types of contributions are welcome. Be sure to read the [Contributor Guides](https://docs.meshery.io/project/contributing) for a tour of resources available to you and how to get started.

## Join the Meshery community!

<a name="contributing"></a><a name="community"></a>
Our projects are community-built and welcome collaboration. 👍 Be sure to see the <a href="https://layer5.io/community/handbook">Contributor Journey Map</a> and <a href="https://layer5.io/community/handbook">Community Handbook</a> for a tour of resources available to you and the <a href="https://layer5.io/community/handbook/repository-overview">Repository Overview</a> for a cursory description of repository by technology and programming language. Jump into community <a href="https://slack.meshery.io">Slack</a> or <a href="http://discuss.meshery.io">discussion forum</a> to participate.

<p style="clear:both;">
<a href ="https://layer5.io/community"><img alt="MeshMates" src=".github/assets/images/readme/layer5-community-sign.png" style="margin-right:36px; margin-bottom:7px;" width="140px" align="left" /></a>
<h3>Find your MeshMate</h3>

<p>MeshMates are experienced Layer5 community members, who will help you learn your way around, discover live projects, and expand your community network. Conneect with a Meshmate today!</p>

Find out more on the <a href="https://layer5.io/community#meshmate">Layer5 community</a>. <br />

</p>
<br /><br />
<div style="display: flex; justify-content: center; align-items:center;">
<div>
<a href="https://meshery.io/community"><img alt="Layer5 Cloud Native Community" src="https://docs.meshery.io/assets/img/readme/community.png" width="140px" style="margin-right:36px; margin-bottom:7px;" width="140px" align="left"/></a>
</div>
<div style="width:60%; padding-left: 16px; padding-right: 16px">
<p>
✔️ <em><strong>Join</strong></em> any or all of the weekly meetings on <a href="https://meshery.io/calendar">community calendar</a>.<br />
✔️ <em><strong>Watch</strong></em> community <a href="https://www.youtube.com/playlist?list=PL3A-A6hPO2IMPPqVjuzgqNU5xwnFFn3n0">meeting recordings</a>.<br />
✔️ <em><strong>Fill-in</strong></em> a <a href="https://layer5.io/newcomers">community member form</a> to gain access to community resources.
<br />
✔️ <em><strong>Discuss</strong></em> in the <a href="https://discuss.layer5.io">Community Forum</a>.<br />
✔️ <em><strong>Explore more</strong></em> in the <a href="https://layer5.io/community/handbook">Community Handbook</a>.<br />
</p>
</div><br /><br />
<div>
<a href="http://slack.layer5.io">
<picture style="text-align:left;" align="left">
  <source media="(prefers-color-scheme: dark)" srcset="https://raw.githubusercontent.com/meshery/meshery/master/.github/assets/images/readme/slack.svg"  width="110px" />
  <source media="(prefers-color-scheme: light)" srcset="https://raw.githubusercontent.com/meshery/meshery/master/.github/assets/images/readme/slack.svg" width="110px" />
  <img alt="Shows an illustrated light mode meshery logo in light color mode and a dark mode meshery logo dark color mode." src="https://raw.githubusercontent.com/meshery/meshery/master/.github/assets/images/readme/slack.svg" width="110px" align="left" />
</picture>
</a>
  <br /><br />
  <p align="left">
<i>Not sure where to start?</i> Grab an open issue with the <a href="https://github.com/issues?q=is%3Aopen+is%3Aissue+archived%3Afalse+org%3Alayer5io+org%3Ameshery+org%3Aservice-mesh-performance+org%3Aservice-mesh-patterns+label%3A%22help+wanted%22+">help-wanted label</a>.
</p>
</div><br /><br />
  
<div>&nbsp;</div>



<!-- <a href="https://youtu.be/MXQV-i-Hkf8"><img alt="Deploying Linkerd with Meshery" src="https://docs.meshery.io/assets/img/readme/deploying-linkerd-with-meshery.png" width="100%" align="center" /></a> -->


### License

This repository and site are available as open-source under the terms of the [Apache 2.0 License](https://opensource.org/licenses/Apache-2.0).
